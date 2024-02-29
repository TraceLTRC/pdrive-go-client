package main

import (
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/viper"
	"github.com/traceltrc/pdrive-go-client/internal/shared"
	"github.com/traceltrc/pdrive-go-client/internal/shared/constants"
	"github.com/vbauerster/mpb/v8"
)

func main() {
	// Init config
	shared.InitConfig()
	token := viper.GetString(string(constants.TOKEN))
	api_url_string := viper.GetString(string(constants.API_URL))
	api_url, err := url.Parse(api_url_string)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse URL from config: %v", err)
		return
	}

	args := os.Args
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr,
			"Error: No filepath was given\nCorrect command usage: pdrive-go-client [FILE]",
		)
		return
	}

	pathfile := os.Args[1]
	stat, err := os.Stat(pathfile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to get filesize: %v", err)
		return
	}

	progress := mpb.New()

	size := stat.Size()
	if size > constants.SPLIT_SIZE {
		file_url, err := shared.MultiUpload(pathfile, api_url, token, 2, progress, size)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			return
		}
		fmt.Println(file_url)
	} else {
		file_url, err := shared.UploadSingle(pathfile, api_url, token, progress, size)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
			return
		}
		fmt.Println(file_url)
	}
}
