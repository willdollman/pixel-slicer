package pixelio

// TODO: Should this be a submodule/folder of pixelslicer? How does namespace work here?
// Ideally I'd call this io, but that's already taken!

import (
	"log"
	"os"
	"path/filepath"
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
