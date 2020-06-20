package pixelio

// TODO: Should this be a submodule/folder of pixelslicer? How does namespace work here?
// Ideally I'd call this io, but that's already taken!

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type InputFile struct {
	Path     string // Absolute filesystem path to file
	Filename string // Name of file with extension
	Subdir   string // Subdirectory relative to input directory
}

// EnumerateDirContents enumerates the contents of a directory, returning
// an array of inputFiles
func EnumerateDirContents(dir string) (files []InputFile, err error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if info == nil {
			return errors.New("File path does not exist: " + path)
		}

		if info.IsDir() {
			return nil
		}

		// want to know how to refer to the file using the path string that's passed in
		// but ALSO want to know what subdirectory the file is. But what do we want that
		// subdirectory to be relative to? Presumably the actual directory that was passed in, right?

		// So want the full path of the passed-in directory AND the file, and then get the relative directory

		// Work out the file's path relative to the input directory
		relPath, err := filepath.Rel(absDir, path)
		if err != nil {
			return err
		}

		subdir, filename := filepath.Split(relPath)

		file := InputFile{
			Path:     path,
			Filename: filename,
			Subdir:   subdir,
		}
		files = append(files, file)
		return nil
	})
	return
}

// FilterFileType filters lists of files by type - image, video, etc
func FilterFileType(files []InputFile, fileType string) (filteredFiles []InputFile) {
	typeExtensions := make(map[string][]string)
	typeExtensions["image"] = []string{".jpg", ".jpeg", ".png", ".tiff"}
	typeExtensions["video"] = []string{".mp4", ".mov"}

	validExtensions, ok := typeExtensions[fileType]
	if ok == false {
		log.Fatalf("No file type '%s'", fileType)
	}

FILE:
	for _, file := range files {
		for _, validExtension := range validExtensions {
			if strings.ToLower(filepath.Ext(file.Path)) == validExtension {
				filteredFiles = append(filteredFiles, file)
				continue FILE
			}
		}
	}
	return
}

// TODO: Load from config file
func baseOutputDir() (baseOutputDir string) {
	return "output"
}

// GetFileOutputDir returns the path of the output dir for a given file
func GetFileOutputDir(f InputFile) (outputDir string) {
	outputDir = filepath.Join(baseOutputDir(), f.Subdir)
	return
}

// GetFileOutputPath returns the path of the output version of a given file, included modifying the file extension
func GetFileOutputPath(f InputFile, size int, ext string) (outputPath string) {
	outputFilename := strings.TrimSuffix(f.Filename, filepath.Ext(f.Filename)) + "-" + strconv.Itoa(size) + "." + ext

	outputPath = filepath.Join(GetFileOutputDir(f), outputFilename)

	return
}

// EnsureOutputDirExists ensures that the configured output dir, or subdirectory thereof, exists
func EnsureOutputDirExists(subdir string) error {
	// Top level output dir
	// TODO: Configure in config
	dirPermissions := os.FileMode(0755)

	outputDir := filepath.Join(baseOutputDir(), subdir)
	f, err := os.Stat(outputDir)
	// If dir doesn't exist, create it
	if os.IsNotExist(err) {
		if err := os.Mkdir(outputDir, dirPermissions); err != nil {
			return err
		}
	} else {
		// If file with outputDir's name exists, ensure it is really a directory
		if !f.IsDir() {
			fmt.Println("Not a directory", outputDir)
			// err := MediaprocessorError{s: "Output subdirectory exists as a file"}
			// fmt.Println("Err:", err.s)
			return errors.New("Output subdirectory exists as a file")
		}
		if f.Mode() != dirPermissions {
			if err := os.Chmod(outputDir, dirPermissions); err != nil {
				return errors.New("Unable to update permissions on output dir")
			}
		}
	}

	return nil
}
