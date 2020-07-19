package pixelio

// TODO: Should this be a submodule/folder of pixelslicer? How does namespace work here?
// Ideally I'd call this io, but that's already taken!

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/willdollman/pixel-slicer/internal/pixelslicer/config"
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

func TypeExtension() map[string][]string {
	return map[string][]string{
		"image": {".jpg", ".jpeg", ".png", ".tiff"},
		"video": {".mp4", ".mov"},
	}
}

// ExtensionType inverts TypeExtension to simplify file extension lookups
func ExtensionType() map[string]string {
	et := make(map[string]string)
	for mediaType, extensions := range TypeExtension() {
		for _, extension := range extensions {
			et[extension] = mediaType
		}
	}
	return et
}

// GetMediaType detects media type from a file's extension.
// Returns the corresponding key from TypeExtension()
func GetMediaType(file InputFile) (MediaType string) {
	fileExt := strings.ToLower(filepath.Ext(file.Path))
	mediaType, ok := ExtensionType()[fileExt]
	if !ok {
		fmt.Println("Invalid media type for", file.Path)
		return
	}
	return mediaType
}

// FilterValidFiles returns all valid file types in the input
func FilterValidFiles(files []InputFile) (filteredFiles []InputFile) {
	for mediaType, _ := range TypeExtension() {
		f := FilterFileType(files, mediaType)
		filteredFiles = append(filteredFiles, f...)
	}
	return
}

// FilterFileType filters lists of files by type - image, video, etc
func FilterFileType(files []InputFile, fileType string) (filteredFiles []InputFile) {
	validExtensions, ok := TypeExtension()[fileType]
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
// TODO: Do I want to something that's compatible with a Image/VideoConfiguration instead of width and ext?
func GetFileOutputPath(f InputFile, mediaConfig config.MediaConfiguration) (outputPath string) {
	// Include image quality in filename for debugging
	outputFilename := strings.TrimSuffix(f.Filename, filepath.Ext(f.Filename)) + mediaConfig.OutputFileName(false)
	// outputFilename := strings.TrimSuffix(f.Filename, filepath.Ext(f.Filename)) + "-" + strconv.Itoa(config.MaxWidth) + "-" + strconv.Itoa(config.Quality) + "." + ext

	outputPath = filepath.Join(GetFileOutputDir(f), outputFilename)

	return
}

// EnsureOutputDirExists ensures that the configured output dir, or subdirectory thereof, exists
func EnsureOutputDirExists(subdir string) error {
	fullDir := filepath.Join(baseOutputDir(), subdir)

	return EnsureDirExists(fullDir)
}

// EnsureDirExists ensures that given path exists, is a directory, and has the correct permissions
func EnsureDirExists(dir string) error {
	// Top level output dir
	// TODO: Configure in config
	dirPermissions := os.FileMode(0755)

	f, err := os.Stat(dir)
	// If dir doesn't exist, create it
	if os.IsNotExist(err) {
		if err := os.MkdirAll(dir, dirPermissions); err != nil {
			return err
		}
	} else {
		// If file with outputDir's name exists, ensure it is really a directory
		if !f.IsDir() {
			fmt.Println("Not a directory", dir)
			// err := MediaprocessorError{s: "Output subdirectory exists as a file"}
			// fmt.Println("Err:", err.s)
			return errors.New("Output subdirectory exists as a file")
		}
		if f.Mode() != dirPermissions {
			if err := os.Chmod(dir, dirPermissions); err != nil {
				return errors.New("Unable to update permissions on output dir")
			}
		}
	}

	return nil
}

// MoveOriginal moves the provided input file to the provided directory, preserving subdirectory structure
func MoveOriginal(file InputFile, moveDir string) (err error) {
	fullMoveDir := filepath.Join(moveDir, file.Subdir)
	if err = EnsureDirExists(fullMoveDir); err != nil {
		return err
	}

	movedFileName := filepath.Join(fullMoveDir, file.Filename)
	fmt.Printf("Moving file %s to %s\n", file.Path, movedFileName)

	// Remove target output file if it already exists
	if _, err := os.Stat(movedFileName); err == nil {
		fmt.Println("File already exists - removing")
		if err = os.Remove(movedFileName); err != nil {
			return err
		}
	}

	// Move input file
	if err = os.Rename(file.Path, movedFileName); err != nil {
		return err
	}

	return nil
}
