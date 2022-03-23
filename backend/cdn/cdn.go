// Package cdn provides some functions to interface with the firebase bucket
package cdn

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"

	"github.com/soumitradev/Dwitter/backend/auth"
	"github.com/soumitradev/Dwitter/backend/common"
	"github.com/soumitradev/Dwitter/backend/util"

	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go/v4"
	"github.com/disintegration/imaging"
	"github.com/golang/gddo/httputil/header"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func init() {
	// Connect to bucket
	opt := option.WithCredentialsFile("backend/cdn_key.json")
	config := &firebase.Config{
		StorageBucket: "dwitter-72e9d.appspot.com",
	}

	app, err := firebase.NewApp(common.BaseCtx, config, opt)
	if err != nil {
		panic(fmt.Errorf("error initializing app: %v", err))
	}

	fireClient, err := app.Storage(common.BaseCtx)
	if err != nil {
		panic(fmt.Errorf("error initializing client: %v", err))
	}

	common.Bucket, err = fireClient.DefaultBucket()
	if err != nil {
		panic(fmt.Errorf("error initializing common.Bucket: %v", err))
	}
}

// Convert a link to just it's stem, like "media/:id"
func LinkToLocation(link string) (string, error) {
	// Ckeck if link matches the format of a link
	linkRegex, err := regexp.Compile(`^https://storage\.googleapis\.com/download/storage/v1/b/dwitter\-72e9d\.appspot\.com/o/\w+\%2F\w+\.\w+\?.+$`)
	if err != nil {
		return "", err
	}
	matched := linkRegex.MatchString(link)
	if matched {
		// Reduce the link to its stem
		prefixRegex, err := regexp.Compile(`^https://storage\.googleapis\.com/download/storage/v1/b/dwitter\-72e9d\.appspot\.com/o/`)
		if err != nil {
			return "", err
		}
		noPrefix := prefixRegex.ReplaceAllString(link, "")
		queryRegex, err := regexp.Compile(`\?.+$`)
		if err != nil {
			return "", err
		}
		stem := queryRegex.ReplaceAllString(noPrefix, "")

		slashRegex, err := regexp.Compile(`\%2F`)
		if err != nil {
			return "", err
		}
		finalLink := slashRegex.ReplaceAllString(stem, "/")

		return finalLink, nil
	} else {
		return "", errors.New("media link invalid")
	}
}

// Delete an object given a stem, and allow deletion of its thumbnail
func DeleteLocation(location string, deleteThumb bool) error {
	// Delete corresponding thumbnail
	if deleteThumb {
		mediaCheckRegex, err := regexp.Compile(`media\/`)
		if err != nil {
			return err
		}
		thumbLocNoFormat := mediaCheckRegex.ReplaceAllString(location, "thumb/")

		extensionRegex, err := regexp.Compile(`\.\w+$`)
		if err != nil {
			return err
		}
		thumbLoc := extensionRegex.ReplaceAllString(thumbLocNoFormat, ".png")

		o := common.Bucket.Object(thumbLoc)
		if err := o.Delete(common.BaseCtx); err != nil {
			if err.Error() == "storage: object doesn't exist" {
				return errors.New("thumbnail for media not found")
			}
			return fmt.Errorf("Object(%q).Delete: %v", location, err)
		}
	}

	// Delete object
	o := common.Bucket.Object(location)
	if err := o.Delete(common.BaseCtx); err != nil {
		if err.Error() == "storage: object doesn't exist" {
			return errors.New("media not found")
		}
		return fmt.Errorf("Object(%q).Delete: %v", location, err)
	}
	return nil
}

// Handle media upload requests
func UploadMediaHandler(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	authHeader := r.Header.Get("authorization")
	_, err := auth.Authenticate(authHeader)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(common.HTTPError{
			Error: err.Error(),
		})
	} else {
		supportedFormats := map[string]bool{
			"image/gif":  true, // GIF
			"image/jpeg": true, // JPEG
			"image/png":  true, // PNG
			"video/mp4":  true, // MP4
		}
		isImage := map[string]bool{
			"image/gif":  true,  // GIF
			"image/jpeg": true,  // JPEG
			"image/png":  true,  // PNG
			"video/mp4":  false, // MP4
		}

		// Check if content type is "multipart/form-data"
		if r.Header.Get("Content-Type") != "" {
			value, _ := header.ParseValueAndParams(r.Header, "Content-Type")
			if value != "multipart/form-data" {
				msg := "Content-Type header is not multipart/form-data"
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnsupportedMediaType)
				json.NewEncoder(w).Encode(common.HTTPError{
					Error: msg,
				})
				return
			}
		}

		// Limit size to 8*8MB = 64MB
		err := r.ParseMultipartForm(64 << 20)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			json.NewEncoder(w).Encode(common.HTTPError{
				Error: "Files exceed file size limit.",
			})
			return
		}

		formData := r.MultipartForm
		files := formData.File["files"]

		// Enforce limits
		if len(files) > 8 {
			msg := "Too many files. Limit is 8 files."
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			json.NewEncoder(w).Encode(common.HTTPError{
				Error: msg,
			})
			return
		}

		for _, file := range files {
			if file.Size > (8 << 20) {
				msg := "File too large. Limit is 8 files, 8MB each."
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				json.NewEncoder(w).Encode(common.HTTPError{
					Error: msg,
				})
				return
			}
			if !supportedFormats[file.Header.Get("Content-Type")] {
				msg := "Format unsupported."
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnsupportedMediaType)
				json.NewEncoder(w).Encode(common.HTTPError{
					Error: msg,
				})
				return
			}
		}

		links := []string{}

		for i := range files {
			// Open file in request
			file, err := files[i].Open()
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(common.HTTPError{
					Error: err.Error(),
				})
				return
			}

			defer file.Close()

			operationCtx, cancel := context.WithTimeout(common.BaseCtx, time.Second*50)
			defer cancel()

			// Upload an object with storage.Writer.

			// Get a unique name for object
			randID := util.GenID(30)
			found := true
			for found {
				query := &storage.Query{Prefix: randID}
				it := common.Bucket.Objects(operationCtx, query)
				numObj := 0
				for {
					_, err := it.Next()
					if err == iterator.Done {
						if numObj == 0 {
							found = false
							break
						}
					}
					if err != nil {
						log.Fatal(err)
					}
					numObj += 1
					break
				}
				if numObj != 0 {
					randID = util.GenID(30)
				}
			}

			// Write file to cloud
			obj := common.Bucket.Object("media/" + randID + filepath.Ext(files[i].Filename))
			writer := obj.NewWriter(operationCtx)
			if _, err = io.Copy(writer, file); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(common.HTTPError{
					Error: err.Error(),
				})
				return
			}
			if err := writer.Close(); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(common.HTTPError{
					Error: err.Error(),
				})
				return
			}

			var thumb *image.NRGBA
			_, err = file.Seek(0, 0)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(common.HTTPError{
					Error: err.Error(),
				})
				return
			}

			// Make thumbnail based on image or video
			if isImage[files[i].Header.Get("Content-Type")] {
				// Make thumbnail
				image, _, err := image.Decode(file)
				if err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(common.HTTPError{
						Error: err.Error(),
					})
					return
				}

				// If wider than tall, fit to height
				xSize := image.Bounds().Dx()
				ySize := image.Bounds().Dy()
				if xSize > ySize {
					newWidth := (xSize / ySize) * 640
					thumb = imaging.Thumbnail(image, newWidth, 360, imaging.NearestNeighbor)
				} else {
					// Else, fit to width
					newHeight := (ySize / xSize) * 360
					thumb = imaging.Thumbnail(image, 640, newHeight, imaging.NearestNeighbor)
				}

			} else {
				videoBytes := make([]byte, files[i].Size)
				file.Read(videoBytes)

				thumbnailBytes, err := generateVideoThumbnail(videoBytes)
				if err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(common.HTTPError{
						Error: err.Error(),
					})
					return
				}

				buf := bytes.NewBuffer(thumbnailBytes)
				picDat, err := png.Decode(buf)
				if err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(common.HTTPError{
						Error: err.Error(),
					})
					return
				}

				// If wider than tall, fit to height
				xSize := picDat.Bounds().Dx()
				ySize := picDat.Bounds().Dy()
				if xSize > ySize {
					newWidth := (xSize / ySize) * 640
					thumb = imaging.Thumbnail(picDat, newWidth, 360, imaging.NearestNeighbor)
				} else {
					// Else, fit to width
					newHeight := (ySize / xSize) * 360
					thumb = imaging.Thumbnail(picDat, 640, newHeight, imaging.NearestNeighbor)
				}
			}

			// Save thumbnail
			thumbObj := common.Bucket.Object("thumb/" + randID + ".png")
			thumbWriter := thumbObj.NewWriter(operationCtx)
			if err = png.Encode(thumbWriter, thumb); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(common.HTTPError{
					Error: err.Error(),
				})
				panic(fmt.Errorf("png.Encode: %v", err))
			}
			if err := thumbWriter.Close(); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(common.HTTPError{
					Error: err.Error(),
				})
				panic(fmt.Errorf("png.Encode: %v", err))
			}

			mediaLink := writer.Attrs().MediaLink
			common.MediaCreatedButNotUsed[mediaLink] = true

			go destroyObjectAfterExpire(10, mediaLink)

			links = append(links, mediaLink)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(links)
		return
	}
}

// Handle pfp upload requests
func UploadPFPHandler(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	// authHeader := r.Header.Get("authorization")
	cookie, _ := r.Cookie("session")
	var username string
	if cookie != nil {
		cookieString := cookie.Value
		_, err := auth.Authenticate(cookieString)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(common.HTTPError{
				Error: err.Error(),
			})
		}
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(common.HTTPError{
			Error: "No cookie",
		})
	}

	supportedFormats := map[string]bool{
		"image/gif":  true, // GIF
		"image/jpeg": true, // JPEG
		"image/png":  true, // PNG
	}

	// Check if content type is "multipart/form-data"
	if r.Header.Get("Content-Type") != "" {
		value, _ := header.ParseValueAndParams(r.Header, "Content-Type")
		if value != "multipart/form-data" {
			msg := "Content-Type header is not multipart/form-data"
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnsupportedMediaType)
			json.NewEncoder(w).Encode(common.HTTPError{
				Error: msg,
			})
			return
		}
	}

	// Limit size to 8MB
	err := r.ParseMultipartForm(8 << 20)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		json.NewEncoder(w).Encode(common.HTTPError{
			Error: "File exceeds 8MB file size limit.",
		})
		return
	}

	formData := r.MultipartForm
	files := formData.File["files"]

	// Enforce limits
	if len(files) > 1 {
		msg := "Too many files. Limit is 1 file."
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		json.NewEncoder(w).Encode(common.HTTPError{
			Error: msg,
		})
		return
	}

	file := files[0]
	if file.Size > (8 << 20) {
		msg := "File too large. Limit is 8MB."
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		json.NewEncoder(w).Encode(common.HTTPError{
			Error: msg,
		})
		return
	}
	if !supportedFormats[file.Header.Get("Content-Type")] {
		msg := "Format unsupported."
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnsupportedMediaType)
		json.NewEncoder(w).Encode(common.HTTPError{
			Error: msg,
		})
		return
	}

	links := []string{}

	for i := range files {
		// Open file
		file, err := files[i].Open()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(common.HTTPError{
				Error: err.Error(),
			})
			return
		}
		defer file.Close()

		operationCtx, cancel := context.WithTimeout(common.BaseCtx, time.Second*50)
		defer cancel()

		// Upload to cloud
		randID := util.GenID(30)
		found := true
		for found {
			query := &storage.Query{Prefix: randID}
			it := common.Bucket.Objects(operationCtx, query)
			numObj := 0
			for {
				_, err := it.Next()
				if err == iterator.Done {
					if numObj == 0 {
						found = false
						break
					}
				}
				if err != nil {
					log.Fatal(err)
				}
				numObj += 1
				break
			}
			if numObj != 0 {
				randID = util.GenID(30)
			}
		}

		decoded, _, err := image.Decode(file)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(common.HTTPError{
				Error: err.Error(),
			})
			return
		}

		// If wider than tall, fit to height
		thumb := imaging.Thumbnail(decoded, 240, 240, imaging.NearestNeighbor)

		obj := common.Bucket.Object("pfp/pfp_" + username + filepath.Ext(files[i].Filename))
		writer := obj.NewWriter(operationCtx)
		if err = png.Encode(writer, thumb); err != nil {
			panic(fmt.Errorf("io.Copy: %v", err))
		}
		if err := writer.Close(); err != nil {
			panic(fmt.Errorf("io.Copy: %v", err))
		}
		links = append(links, writer.Attrs().MediaLink)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(links)
	return
}

// Generate a thumbnail from a video in the tmp directory
func generateVideoThumbnail(videoBytes []byte) ([]byte, error) {
	// command line args, path, and command
	command := "ffmpeg"
	frameExtractionTime := "0:00:00.000"
	vframes := "1"
	cv := "png"
	format := "image2pipe"

	cmd := exec.Command(
		command,
		"-i", "pipe:0", // read from stdin
		"-ss", frameExtractionTime,
		"-vframes", vframes,
		"-c:v", cv,
		"-f", format,
		"pipe:1",
	)

	pipeIn, _ := cmd.StdinPipe()
	writer := bufio.NewWriter(pipeIn)
	pipeOut, _ := cmd.StdoutPipe()
	cmd.Start()

	go func() {
		defer writer.Flush()
		defer pipeIn.Close()
		writer.Write(videoBytes)
	}()

	defer pipeOut.Close()
	imageBytes, err := io.ReadAll(pipeOut)
	if err != nil {
		return nil, err
	}

	return imageBytes, nil
}

// Destroy an object when it expires
func destroyObjectAfterExpire(minutes int, id string) {
	time.Sleep(time.Minute * time.Duration(minutes))
	if common.MediaCreatedButNotUsed[id] {
		loc, err := LinkToLocation(id)
		if err != nil {
			fmt.Printf("Error finding media: %v", err)
		}
		err = DeleteLocation(loc, true)
		if err != nil {
			fmt.Printf("Error auto-deleting media: %v", err)
		}
	}
}
