package main

import (
	"image/jpeg"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/harry1453/go-common-file-dialog/cfd"
	"github.com/jdeng/goheif"
)

func main() {

	files := selectFiles()
	outFolder := selectFolder()

	for _, fin := range files {
		fout := filepath.Join(outFolder, filepath.Base(fin[:len(fin)-len(filepath.Ext(fin))]+".jpeg"))
		log.Println(fout)
		fi, err := os.Open(fin)
		if err != nil {
			log.Fatal(err)
		}
		defer fi.Close()

		exif, err := goheif.ExtractExif(fi)
		if err != nil {
			log.Printf("Warning: no EXIF from %s: %v\n", fin, err)
		}

		img, err := goheif.Decode(fi)
		if err != nil {
			log.Fatalf("Failed to parse %s: %v\n", fin, err)
		}

		fo, err := os.OpenFile(fout, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			log.Fatalf("Failed to create output file %s: %v\n", fout, err)
		}
		defer fo.Close()

		w, _ := newWriterExif(fo, exif)
		err = jpeg.Encode(w, img, nil)
		if err != nil {
			log.Fatalf("Failed to encode %s: %v\n", fout, err)
		}

		log.Printf("Convert %s to %s successfully\n", fin, fout)
	}
}

func selectFiles() []string {
	openMultiDialog, err := cfd.NewOpenMultipleFilesDialog(cfd.DialogConfig{
		Title: "Выберите файлы для перекодировки",
		Role:  "OpenFilesExample",
		FileFilters: []cfd.FileFilter{
			{
				DisplayName: "HEIC images (*.heic)",
				Pattern:     "*.heic",
			},
			{
				DisplayName: "All Files (*.*)",
				Pattern:     "*.*",
			},
		},
		SelectedFileFilterIndex: 0,
		FileName:                "",
		DefaultExtension:        "heic",
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := openMultiDialog.Show(); err != nil {
		log.Fatal(err)
	}
	results, err := openMultiDialog.GetResults()
	if err == cfd.ErrorCancelled {
		log.Fatal("Dialog was cancelled by the user.")
	} else if err != nil {
		log.Fatal(err)
	}
	log.Printf("Chosen file(s): %s\n", results)
	return results
}

func selectFolder() string {
	pickFolderDialog, err := cfd.NewSelectFolderDialog(cfd.DialogConfig{
		Title: "Выбери папку для сохранения перекодированных файлов",
		Role:  "PickFolderExample",
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := pickFolderDialog.Show(); err != nil {
		log.Fatal(err)
	}
	result, err := pickFolderDialog.GetResult()
	if err == cfd.ErrorCancelled {
		log.Fatal("Dialog was cancelled by the user.")
	} else if err != nil {
		log.Fatal(err)
	}
	log.Printf("Chosen folder: %s\n", result)
	return result
}

// Skip Writer for exif writing
type writerSkipper struct {
	w           io.Writer
	bytesToSkip int
}

func (w *writerSkipper) Write(data []byte) (int, error) {
	if w.bytesToSkip <= 0 {
		return w.w.Write(data)
	}

	if dataLen := len(data); dataLen < w.bytesToSkip {
		w.bytesToSkip -= dataLen
		return dataLen, nil
	}

	if n, err := w.w.Write(data[w.bytesToSkip:]); err == nil {
		n += w.bytesToSkip
		w.bytesToSkip = 0
		return n, nil
	} else {
		return n, err
	}
}

func newWriterExif(w io.Writer, exif []byte) (io.Writer, error) {
	writer := &writerSkipper{w, 2}
	soi := []byte{0xff, 0xd8}
	if _, err := w.Write(soi); err != nil {
		return nil, err
	}

	if exif != nil {
		app1Marker := 0xe1
		markerlen := 2 + len(exif)
		marker := []byte{0xff, uint8(app1Marker), uint8(markerlen >> 8), uint8(markerlen & 0xff)}
		if _, err := w.Write(marker); err != nil {
			return nil, err
		}

		if _, err := w.Write(exif); err != nil {
			return nil, err
		}
	}

	return writer, nil
}
