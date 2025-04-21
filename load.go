package hangover

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// cspell:words nqauds

var (
	errWrongArgCount  = errors.New("need one or two arguments")
	errIndexPassed    = errors.New("you should provide a path to an xml and nq file, or a directory providing both")
	errNotRegularFile = errors.New("not a regular file")
	errNeedOneFile    = errors.New("need exactly one '*.xml' and one '*.nq' file")
)

// FindSource finds the sources for the given path.
// FindSource does not guarantee that contents are loadable.
func FindSource(argv ...string) (nq, xml string, err error) {
	if len(argv) == 0 || len(argv) > 2 {
		return "", "", errWrongArgCount
	}

	// two arguments provided: use xml, then nqauds
	if len(argv) == 2 {
		nq = argv[1]
		xml = argv[0]
	} else {
		isDir, err := isDirectory(argv[0])
		if err != nil {
			return "", "", err
		}

		// try to read the index
		if !isDir {
			return "", "", errIndexPassed
		}

		base := argv[0]

		xmls, err := filepath.Glob(filepath.Join(base, "*.xml"))
		if err != nil {
			return "", "", fmt.Errorf("failed to list xml files: %w", err)
		}
		if len(xmls) != 1 {
			return "", "", fmt.Errorf("%w: found exactly %d '.xml' files in %q", errNeedOneFile, len(xml), base)
		}
		xml = xmls[0]

		nqs, err := filepath.Glob(filepath.Join(base, "*.nq"))
		if err != nil {
			return "", "", fmt.Errorf("failed to list nq files: %w", err)
		}
		if len(nqs) != 1 {
			return "", "", fmt.Errorf("%w: found exactly %d '.nq' files in %q", errNeedOneFile, len(nqs), base)
		}
		nq = nqs[0]
	}

	// check for regular files
	for _, file := range [2]string{nq, xml} {
		ok, err := isFile(file)
		if err != nil {
			return "", "", fmt.Errorf("failed to check file status: %w", err)
		}
		if !ok {
			return "", "", fmt.Errorf("%w: %q", errNotRegularFile, file)
		}
	}

	return nq, xml, nil
}

func isDirectory(path string) (ok bool, err error) {
	stats, err := os.Stat(path)
	if err != nil {
		return false, fmt.Errorf("failed to stat %q: %w", path, err)
	}
	return stats.Mode().IsDir(), nil
}

// isFile checks if path is a.
func isFile(path string) (ok bool, err error) {
	stats, err := os.Stat(path)
	if err != nil {
		return false, fmt.Errorf("failed to stat %q: %w", path, err)
	}
	return stats.Mode().IsRegular(), nil
}
