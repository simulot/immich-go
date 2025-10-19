package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/disintegration/imaging"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/simulot/immich-go/internal/worker"
	"gopkg.in/yaml.v3"
)

type CategoryMembersResponse struct {
	Query struct {
		Categorymembers []struct {
			Title string `json:"title"`
		} `json:"categorymembers"`
	} `json:"query"`
	Continue struct {
		Cmcontinue string `json:"cmcontinue"`
	} `json:"continue"`
}

type ImageInfoResponse struct {
	Query struct {
		Pages map[string]struct {
			Imageinfo []struct {
				Url string `json:"url"`
			} `json:"imageinfo"`
		} `json:"pages"`
	} `json:"query"`
}

type MetadataResponse struct {
	Query struct {
		Pages map[string]struct {
			Imageinfo []struct {
				Extmetadata map[string]interface{} `json:"extmetadata"`
			} `json:"imageinfo"`
		} `json:"pages"`
	} `json:"query"`
}

type ImageData struct {
	URL      string
	Title    string
	Metadata map[string]interface{}
}

func main() {
	err := run()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func run() error {
	w := worker.NewPool(4)
	defer w.Stop()
	categories := []string{
		"Insects",
		"Mountains",
		"Musical instruments",
		"Telescopes",
		"Bridges",
		"Horses",
	}

	client := &http.Client{}
	var totalImages atomic.Int32
	destination := "../wikimedia"
	err := os.MkdirAll(destination, 0o755)
	if err != nil {
		return err
	}

	for _, category := range categories {
		images := getCategoryImages(client, category, 10)
		for i, img := range images {
			w.Submit(func() {
				filename := path.Join(destination, fmt.Sprintf("%s_%02d.jpg", strings.ReplaceAll(strings.ToLower(category), " ", "_"), i+1))
				err := downloadImage(client, img.URL, filename, img.Title, img.Metadata)
				if err != nil {
					fmt.Printf("Error downloading %s: %v\n", filename, err)
				} else {
					fmt.Printf("Downloaded %s\n", filename)
					totalImages.Add(1)
				}
			})
		}
	}
	w.Stop()
	fmt.Printf("Total images downloaded: %d\n", totalImages.Load())
	return nil
}

func getCategoryImages(client *http.Client, category string, count int) []ImageData {
	var images []ImageData
	apiURL := fmt.Sprintf("https://commons.wikimedia.org/w/api.php?action=query&list=categorymembers&cmtype=file&cmtitle=Category:%s&cmlimit=50&format=json", url.QueryEscape(category))

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", "ImmichGoDownloader/1.0 (https://github.com/simulot/immich-go)")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error fetching category %s: %v\n", category, err)
		return images
	}
	defer resp.Body.Close()

	var cmResp CategoryMembersResponse
	_ = json.NewDecoder(resp.Body).Decode(&cmResp)

	for _, member := range cmResp.Query.Categorymembers {
		if len(images) >= count {
			break
		}
		if isImageFile(member.Title) {
			data := getImageURL(client, member.Title)
			if data != nil {
				images = append(images, *data)
			}
		}
	}
	return images
}

func isImageFile(title string) bool {
	ext := strings.ToLower(filepath.Ext(title))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif"
}

func getImageURL(client *http.Client, title string) *ImageData {
	// Get URL
	apiURL := fmt.Sprintf("https://commons.wikimedia.org/w/api.php?action=query&prop=imageinfo&iiprop=url&titles=%s&format=json", url.QueryEscape(title))

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", "ImmichGoDownloader/1.0 (https://github.com/simulot/immich-go)")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var iiResp ImageInfoResponse
	_ = json.NewDecoder(resp.Body).Decode(&iiResp)

	var imgURL string
	for _, page := range iiResp.Query.Pages {
		if len(page.Imageinfo) > 0 {
			imgURL = page.Imageinfo[0].Url
			break
		}
	}
	if imgURL == "" {
		return nil
	}

	// Get metadata
	apiURL2 := fmt.Sprintf("https://commons.wikimedia.org/w/api.php?action=query&prop=imageinfo&iiprop=extmetadata&titles=%s&format=json", url.QueryEscape(title))

	req2, _ := http.NewRequest("GET", apiURL2, nil)
	req2.Header.Set("User-Agent", "ImmichGoDownloader/1.0 (https://github.com/simulot/immich-go)")

	resp2, err := client.Do(req2)
	if err != nil {
		return &ImageData{URL: imgURL, Title: title, Metadata: nil}
	}
	defer resp2.Body.Close()

	var mdResp MetadataResponse
	_ = json.NewDecoder(resp2.Body).Decode(&mdResp)

	var metadata map[string]interface{}
	for _, page := range mdResp.Query.Pages {
		if len(page.Imageinfo) > 0 {
			metadata = page.Imageinfo[0].Extmetadata
			break
		}
	}

	return &ImageData{
		URL:      imgURL,
		Title:    title,
		Metadata: metadata,
	}
}

func downloadImage(client *http.Client, imgURL, filename, title string, metadata map[string]interface{}) error {
	req, _ := http.NewRequest("GET", imgURL, nil)
	req.Header.Set("User-Agent", "ImmichGoDownloader/1.0 (https://github.com/simulot/immich-go)")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("status code %d", resp.StatusCode)
	}

	// Read image data
	imgData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Decode image
	img, err := imaging.Decode(bytes.NewReader(imgData))
	if err != nil {
		return err
	}

	// Resize image to fit under 50KB while maintaining aspect ratio
	resizedImg, err := resizeToFitSize(img, 50*1024) // 50KB
	if err != nil {
		return err
	}

	// Save resized image
	err = imaging.Save(resizedImg, filename, imaging.JPEGQuality(85))
	if err != nil {
		return err
	}

	// Extract EXIF and save YAML
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	x, err := exif.Decode(file)
	exifData := ""
	if err != nil {
		fmt.Printf("No EXIF in %s: %v\n", filename, err)
	} else {
		exifData = x.String()
	}

	info := map[string]interface{}{
		"original_title": title,
		"local_filename": filename,
		"url":            imgURL,
		"author":         metadata["Artist"],
		"attribution":    metadata["Attribution"],
		"license":        metadata["LicenseShortName"],
		"exif":           exifData,
	}

	yamlData, err := yaml.Marshal(info)
	if err != nil {
		return err
	}

	yamlFilename := strings.TrimSuffix(filename, filepath.Ext(filename)) + ".yaml"
	return os.WriteFile(yamlFilename, yamlData, 0o644)
}

func resizeToFitSize(img image.Image, maxSize int) (image.Image, error) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate target dimensions for a rough estimate (aim for ~20KB at low quality)
	// This is a heuristic: assume low quality JPEG at small size gives ~20KB
	targetPixels := 400 * 375 // Roughly 400x375 pixels
	scale := float64(targetPixels) / float64(width*height)
	if scale < 1 {
		newWidth := int(float64(width) * scale)
		newHeight := int(float64(height) * scale)
		if newWidth > 0 && newHeight > 0 {
			img = imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)
		}
	}

	// Try very low quality settings
	qualities := []int{15, 10, 5, 1}

	for _, quality := range qualities {
		size, err := getJPEGSize(img, quality)
		if err != nil {
			return nil, err
		}

		if size <= maxSize {
			return img, nil
		}

		// If still too big, reduce size further
		scale := 0.8
		var currentImg image.Image

		for size > maxSize && scale > 0.1 {
			newWidth := int(float64(width) * scale)
			newHeight := int(float64(height) * scale)

			if newWidth < 100 || newHeight < 100 {
				break
			}

			currentImg = imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)
			size, err = getJPEGSize(currentImg, quality)
			if err != nil {
				return nil, err
			}

			if size <= maxSize {
				return currentImg, nil
			}

			scale *= 0.8
		}
	}

	// Return the smallest version we could achieve
	return img, nil
}

func getJPEGSize(img image.Image, quality int) (int, error) {
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
	if err != nil {
		return 0, err
	}
	return buf.Len(), nil
}
