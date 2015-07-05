package util

import (
    "bytes"
    "encoding/json"
    "fmt"
    "github.com/mitchellh/mapstructure"
    "io/ioutil"
    "os"
    "path"
    "reflect"
    "regexp"
)

const configFileSizeLimit = 10 << 20

func DiscoverConfigs(file_or_directory string) (files []string, err error) {
    fi, err := os.Stat(file_or_directory)
    if err != nil {
        return nil, err
    }
    files = make([]string, 0)
    if fi.IsDir() {
        entries, err := ioutil.ReadDir(file_or_directory)
        if err != nil {
            return nil, err
        }
        for _, filename := range entries {
            files = append(files, path.Join(file_or_directory, filename.Name()))
        }
    } else {
        files = append(files, file_or_directory)
    }
    return files, nil
}

func LoadConfig(filename string) (config map[string]interface{}, err error) {
    config_file, err := os.Open(filename)
    if err != nil {
        Emit("Failed to open config file '%s': %s\n", filename, err)
        return
    }

    fi, _ := config_file.Stat()
    if size := fi.Size(); size > (configFileSizeLimit) {
        Emit("config file (%q) size exceeds reasonable limit (%d) - aborting", filename, size)
        return // REVU: shouldn't this return an error, then?
    }
    if fi.Size() == 0 {
        Emit("config file (%q) is empty, skipping", filename)
        return
    }

    buffer := make([]byte, fi.Size())
    _, err = config_file.Read(buffer)
    Emit("%s\n", buffer)

    buffer, err = StripComments(buffer)
    if err != nil {
        Emit("Failed to strip comments from json: %s\n", err)
        return
    }

    err = json.Unmarshal(buffer, &config)
    if err != nil {
        Emit("Failed unmarshalling json: %s\n", err)
        return
    }

    return
}

func StripComments(data []byte) ([]byte, error) {
    data = bytes.Replace(data, []byte("\r"), []byte(""), 0) // Windows
    lines := bytes.Split(data, []byte("\n"))
    filtered := make([][]byte, 0)

    for _, line := range lines {
        match, err := regexp.Match(`^\s*#`, line)
        if err != nil {
            return nil, err
        }
        if !match {
            filtered = append(filtered, line)
        }
    }

    return bytes.Join(filtered, []byte("\n")), nil
}

// Decode from src to result
func DecodeStruct(result interface{}, src map[string]interface{}) error {
    t := reflect.TypeOf(result)
    if t.Kind() != reflect.Ptr {
        err := fmt.Errorf("Need provide a pointer, instead of %v", result)
        return err
    }
    decoder, err := getDecoder(result)
    if err != nil {
        return err
    }
    return decoder.Decode(src)
}

func getDecoder(result interface{}) (*mapstructure.Decoder, error) {
    return mapstructure.NewDecoder(&mapstructure.DecoderConfig{
        TagName:          "json",
        Result:           result,
        WeaklyTypedInput: false})
}
