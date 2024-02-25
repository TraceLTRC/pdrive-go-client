package shared

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/traceltrc/pdrive-go-client/internal/shared/utils"
)

func UploadSingle(path string, api_url *url.URL, token string) string {
  file, err := os.Open(path)
  if err != nil {
    utils.ErrorExit("Unable to open file: %v", err)
  }
  defer file.Close()

  filename := filepath.Base(path)

  bearer := "Bearer " + token
  post_url := api_url.JoinPath("upload", filename)
  reader := bufio.NewReader(file)
  client := &http.Client{}

  req, err := http.NewRequest("POST", post_url.String(), reader)
  if err != nil {
    utils.ErrorExit("Unable to create request: %v", err)
  }
  req.Header.Add("Authorization", bearer)
  
  fmt.Println("Uploading file...")
  resp, err := client.Do(req)
  if err != nil {
    utils.ErrorExit("Unable to send request to server: %v", err)
  }
  defer resp.Body.Close()
  
  bodyBytes, err := io.ReadAll(resp.Body)
  if err != nil {
    utils.ErrorExit("Unable to convert response body to bytes: %v", err)
  }
  body := string(bodyBytes) // Should contain key

  switch resp.StatusCode {
    case 400:
      utils.ErrorExit("Bad request: %s", body)
    case 401:
      utils.ErrorExit("Unauthorized response from server, wrong token?")
    case 500:
      utils.ErrorExit("Server error: %s", body)
    case 200:
      break
    default:
      utils.ErrorExit("Unexpected status code (%d): %s", resp.StatusCode, body)
  }

  file_url := api_url.JoinPath(body)
  
  return file_url.String()
}
