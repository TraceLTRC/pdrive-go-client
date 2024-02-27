package main

import (
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/viper"
	"github.com/traceltrc/pdrive-go-client/internal/shared"
	"github.com/traceltrc/pdrive-go-client/internal/shared/utils"
	"github.com/vbauerster/mpb/v8"
)

const SPLIT_SIZE = 50 * 1000 * 1000 //50 MB

func main() {
  // Init config
  shared.InitConfig()
  token := viper.GetString(string(shared.TOKEN))
  api_url_string := viper.GetString(string(shared.API_URL))
  api_url, err := url.Parse(api_url_string)
  if err != nil {
    utils.ErrorExit("Failed to parse URL from config: %v", err)
  }

  args := os.Args
  if len(args) < 2 {
    utils.ErrorExit("Error: No filepath was given\nCorrect command usage: pdrive-go-client [FILE]")
  }

  pathfile := os.Args[1]
  stat, err := os.Stat(pathfile)
  if err != nil {
    utils.ErrorExit("Unable to get filesize: %v", err)
  }

  progress := mpb.New()

  size := stat.Size()
  if size > SPLIT_SIZE {
    panic("unimplemented") 
  } else {
    file_url := shared.UploadSingle(pathfile, api_url, token, progress, size)
    fmt.Println(file_url)
  }
}
