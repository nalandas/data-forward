package sensor

import (
    "fmt"
    "testing"
)

type MockDriver struct {
}

func (m MockDriver) LoadConfig(file_or_directory string) error {
    return nil
}

func TestRegister(t *testing.T) {
    dr := MockDriver{}
    Register("database", dr)
    dr1 := GetDriver("database")
    fmt.Printf("MockDriver: %p --- %p \n", &dr, &dr1)
    if dr1 != dr {
        t.Errorf("Fail to regristry database sensor! %v", dr1)
    }

    findIt := false
    names := Drivers()
    for _, name := range names {
        if name == "database" {
            fmt.Println("Find the database driver.")
            findIt = true
        }
    }
    if !findIt {
        t.Error("Can not find the database driver!")
    }
}
