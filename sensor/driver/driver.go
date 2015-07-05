package driver

// Driver is the interface that must be implemented by a sensor
// driver.
type Driver interface {
    // Load configuration from given file or directory
    LoadConfig(file_or_directory string) error
}
