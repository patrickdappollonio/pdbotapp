package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

func ResolvePath(p string) (string, error) {
	// if path isn't relative, Abs merges with $PWD
	// here so we can simply call this
	first, err := filepath.Abs(p)
	if err != nil {
		return "", fmt.Errorf("error while fetching $CWD for absolute path: %s", err)
	}

	// check if the path exists
	if _, err := os.Stat(first); err != nil {
		if os.IsNotExist(err) {
			// if it doesn't exist, let's find the app directory
			// and join with the path passed
			exec, err := os.Executable()
			if err != nil {
				return "", fmt.Errorf("unable to get path to executable: %s", err.Error())
			}

			// remove the binary name of the executable path
			exec = filepath.Dir(exec)

			// exec is already an absolute path
			// so we merge with the path passed
			second := filepath.Join(exec, p)
			if _, err := os.Stat(second); err != nil {
				return "", fmt.Errorf("unable to get stat info of %q: %s", second, err.Error())
			}

			return second, nil
		}

		return "", fmt.Errorf("unable to get stat info of %q: %s", first, err.Error())
	}

	return first, nil
}

func MustResolvePath(p string) string {
	fp, err := ResolvePath(p)
	if err != nil {
		panic(err)
	}

	return fp
}
