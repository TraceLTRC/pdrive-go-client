package shared

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/traceltrc/pdrive-go-client/internal/shared/constants"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

type MultipartInfo struct {
	Key      string
	UploadId string
}

type MultipartPart struct {
	PartNumber int    `json:"partNumber"`
	ETag       string `json:"etag"`
}

type UploadPartResult struct {
	Part  *MultipartPart
	Error error
}

type WorkerInput struct {
	body        io.Reader
	part_url    *url.URL
	token       string
	part_number int
}

func UploadSingle(path string,
	api_url *url.URL,
	token string,
	progress *mpb.Progress,
	size int64) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		msg_err := errors.New("Unable to open file: ")
		return "", errors.Join(msg_err, err)
	}
	defer file.Close()

	bar := progress.AddBar(size,
		mpb.PrependDecorators(decor.Percentage()),
		mpb.AppendDecorators(decor.CountersKibiByte("%d/%d")))

	filename := filepath.Base(path)

	bearer := "Bearer " + token
	post_url := api_url.JoinPath("upload", filename)
	reader := bufio.NewReader(file)
	wrapped_reader := bar.ProxyReader(reader)

	req, err := http.NewRequest("POST", post_url.String(), wrapped_reader)
	if err != nil {
		msg_err := errors.New("Unable to create request: ")
		return "", errors.Join(msg_err, err)
	}
	req.Header.Add("Authorization", bearer)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		msg_err := errors.New("Unable to send request to server: ")
		return "", errors.Join(msg_err, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		msg_err := errors.New("Unable to convert response body to bytes:")
		return "", errors.Join(msg_err, err)
	}
	body := string(bodyBytes) // Should contain key

	switch resp.StatusCode {
	case 400:
		msg_err := fmt.Errorf("Bad request: %s", body)
		return "", msg_err
	case 401:
		msg_err := errors.New("Unauthorized response from server, wrong token?")
		return "", msg_err
	case 500:
		msg_err := fmt.Errorf("Server error: %s", body)
		return "", msg_err
	case 200:
		break
	default:
		msg_err := fmt.Errorf("Unexpected status code (%d): %s", resp.StatusCode, body)
		return "", msg_err
	}
	bar.Wait()

	file_url := api_url.JoinPath(body)
	return file_url.String(), nil
}

func MultiUpload(path string,
	api_url *url.URL,
	token string,
	num_workers int,
	progress *mpb.Progress,
	size int64) (string, error) {

	wg := sync.WaitGroup{}
	filename := filepath.Base(path)

	// Init multipart upload
	fmt.Println("Initializing upload...")
	multipart, err := initUpload(filename, api_url, token)
	if err != nil {
		return "", err
	}

	file, err := os.Open(path)
	if err != nil {
		msg_err := errors.New("Cannot open file: ")
		return "", errors.Join(msg_err, err)
	}

	// Initialize worker pool
	workerInput := make(chan *WorkerInput)
	workerOutput := make(chan *UploadPartResult)
	wg.Add(num_workers)

	for i := 0; i < num_workers; i++ {
		go uploadPartWorker(workerInput, workerOutput, &wg)
	}
	go func() {
		wg.Wait()
		close(workerOutput)
	}()

	// Process inputs for worker pool
	start := int64(0)
	part := 1
	totalParts := int(math.Ceil(float64(size) / constants.SPLIT_SIZE))
	inputs := make([]*WorkerInput, 0, totalParts)
	outputs := make([]*MultipartPart, 0, totalParts)
	for start < size {
		// Gets leftover chunks when splitting by 50MB chunks
		partSize := min(size-start, constants.SPLIT_SIZE)

		bar := progress.AddBar(partSize,
			mpb.PrependDecorators(decor.Name(fmt.Sprintf("Part %d", part))),
			mpb.AppendDecorators(decor.CountersKibiByte("%d / %d")))
		reader := io.NewSectionReader(file, start, partSize)
		wrapped_reader := bar.ProxyReader(reader)
		part_url := api_url.JoinPath("upload-part", "put", multipart.Key, multipart.UploadId)
		part_url_query := part_url.Query()
		part_url_query.Add("partNumber", fmt.Sprint(part))
		part_url.RawQuery = part_url_query.Encode()

		input := WorkerInput{
			body:        wrapped_reader,
			part_url:    part_url,
			token:       token,
			part_number: part,
		}
		inputs = append(inputs, &input)

		start = start + partSize
		part++
	}

	go func() {
		for _, input := range inputs {
			workerInput <- input
		}
		close(workerInput)
	}()

	for output := range workerOutput {
		if output.Error != nil {
			// TODO: While returning the error immediately doesn't break anything,
			// the requests are stil ongoing in the background. It is preferable
			// to cancel all requests and then return the error.
			return "", output.Error
		}
		outputs = append(outputs, output.Part)
	}

	// Combine outputs into json array and send for completion
	finish_url := api_url.JoinPath("upload-part", "finish", multipart.Key, multipart.UploadId)

	jsonBytes, err := json.Marshal(outputs)
	if err != nil {
		msg_err := fmt.Errorf("Failed to encode JSON: %v", err)
		return "", msg_err
	}
	jsonReader := bytes.NewReader(jsonBytes)

	req, err := http.NewRequest("POST", finish_url.String(), jsonReader)
	if err != nil {
		msg_err := fmt.Errorf("Failed to create finishing request: %v", err)
		return "", msg_err
	}
  req.Header.Add("Authorization", "Bearer " + token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		msg_err := fmt.Errorf("Failed to execute finishing request: %v", err)
		return "", msg_err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return api_url.JoinPath(multipart.Key).String(), nil
	} else {
		bodyBytes, err := io.ReadAll(resp.Body)
		var body string

		if err != nil {
			body = fmt.Sprintf("Failed to read body: %v", err)
		} else {
			body = string(bodyBytes)
		}
		msg_err := fmt.Errorf("Unexpected status code (%d): %s", resp.StatusCode, body)
		return "", msg_err
	}
}

func uploadPartWorker(
	inputs <-chan *WorkerInput,
	output chan<- *UploadPartResult,
	wg *sync.WaitGroup,
) {
	defer wg.Done()
	for input := range inputs {
		req, err := http.NewRequest("PUT", input.part_url.String(), input.body)
		if err != nil {
			result := UploadPartResult{
				Part:  nil,
				Error: fmt.Errorf("Part %d error'd: %v", input.part_number, err),
			}
			output <- &result
			continue
		}
    req.Header.Add("Authorization", "Bearer " + input.token)

		part, err := func() (*MultipartPart, error) {
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()

      if resp.StatusCode != 200 {
        err = fmt.Errorf("Part %d recieved unexpected status code (%d)", input.part_number, resp.StatusCode)
        return nil, err
      }

			var part MultipartPart
      err = json.NewDecoder(resp.Body).Decode(&part)
			if err != nil {
				return nil, err
			}

			return &part, nil
		}()
		if err != nil {
			msg_err := fmt.Errorf("Failed to execute request or decode JSON: %v", err)
			result := UploadPartResult{Part: nil, Error: msg_err}
			output <- &result
			continue
		}

		result := UploadPartResult{Part: part, Error: nil}
		output <- &result
	}
}

func initUpload(name string, api_url *url.URL, token string) (*MultipartInfo, error) {
	init_url := api_url.JoinPath("upload-part", "init", name)

	req, err := http.NewRequest("POST", init_url.String(), nil)
	if err != nil {
		msg_err := errors.New("Error creating request: ")
		return nil, errors.Join(msg_err, err)
	}
  req.Header.Add("Authorization", "Bearer " + token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		msg_err := errors.New("Error while executing request: ")
		return nil, errors.Join(msg_err, err)
	}
	defer resp.Body.Close()

	var multipart MultipartInfo
	err = json.NewDecoder(resp.Body).Decode(&multipart)
	if err != nil {
		msg_err := errors.New("Failed to parse JSON while initializing multipart upload: %v")
		return nil, errors.Join(msg_err, err)
	}

	return &multipart, nil
}
