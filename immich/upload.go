package immich

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/simulot/immich-go/internal/assets"
)

const (
	TimeFormat    = "2006-01-02T15:04:05Z"
	ctxCallValues = "call-values"
)

func setContextValue(kv map[string]string) serverRequestOption {
	return func(sc *serverCall, req *http.Request) error {
		if sc.err != nil || kv == nil {
			return nil
		}
		sc.ctx = context.WithValue(sc.ctx, ctxCallValues, kv)
		return nil
	}
}

func (ic *ImmichClient) uploadAsset(ctx context.Context, la *assets.Asset, endPoint string, replaceID string) (AssetResponse, error) {
	if ic.dryRun {
		return AssetResponse{
			ID:     uuid.NewString(),
			Status: UploadCreated,
		}, nil
	}

	var ar AssetResponse
	ext := path.Ext(la.OriginalFileName)
	if strings.TrimSuffix(la.OriginalFileName, ext) == "" {
		la.OriginalFileName = "No Name" + ext // fix #88, #128
	}
	if strings.ToUpper(ext) == ".MP" {
		ext = ".MP4" // #405
		la.OriginalFileName = la.OriginalFileName + ".MP4"
	}

	mtype := ic.TypeFromExt(ext)
	switch mtype {
	case "video", "image":
	default:
		// Check if we're trying to upload something that's not a video or image
		return ar, fmt.Errorf("unsupported file type: %s", ext)
	}

	f, err := la.OpenFile()
	if err != nil {
		return ar, err
	}
	defer f.Close()

	s, err := f.Stat()
	if err != nil {
		return ar, err
	}

	callValues := ic.prepareCallValues(la, s, ext, mtype)
	body, pw := io.Pipe()
	m := multipart.NewWriter(pw)

	go func() {
		defer func() {
			m.Close()
			pw.Close()
		}()

		err = ic.writeMultipartFields(m, callValues)
		if err != nil {
			return
		}

		err = ic.writeFilePart(m, f, la.OriginalFileName, mtype)
		if err != nil {
			return
		}

		if la.FromSideCar != nil && strings.HasSuffix(strings.ToLower(la.FromSideCar.File.Name()), ".xmp") {
			err = ic.writeSideCarPart(m, la)
			if err != nil {
				return
			}
		}
	}()

	var errCall error
	switch endPoint {
	case EndPointAssetUpload:
		errCall = ic.newServerCall(ctx, EndPointAssetUpload).
			do(postRequest("/assets", m.FormDataContentType(), setContextValue(callValues), setAcceptJSON(), setBody(body)), responseJSON(&ar))
	case EndPointAssetReplace:
		errCall = ic.newServerCall(ctx, EndPointAssetReplace).
			do(putRequest("/assets/"+replaceID+"/original", setContextValue(callValues), setAcceptJSON(), setContentType(m.FormDataContentType()), setBody(body)), responseJSON(&ar))
	}
	err = errors.Join(err, errCall)

	// In case of error, enhance the error message with file size info
	if err != nil {
		// Get file size from the stat call we already made
		fileSize := s.Size() // Use the file info from the stat call
		
		 // Look for "413" in the error message as a simple way to detect this error type
		errStr := err.Error()
		if strings.Contains(errStr, "413") || strings.Contains(errStr, "Request Entity Too Large") {
			return ar, fmt.Errorf("file '%s' size %.2f MB exceeds server limits (413 Request Entity Too Large): %w", 
				la.OriginalFileName, float64(fileSize)/(1024*1024), err)
		}
		return ar, err
	}

	return ar, nil
}

func (ic *ImmichClient) prepareCallValues(la *assets.Asset, s fs.FileInfo, ext, mtype string) map[string]string {
	callValues := map[string]string{}
	callValues["deviceAssetId"] = fmt.Sprintf("%s-%d", path.Base(la.OriginalFileName), s.Size())
	callValues["deviceId"] = ic.DeviceUUID
	callValues["assetType"] = mtype
	if !la.CaptureDate.IsZero() {
		callValues["fileCreatedAt"] = la.CaptureDate.Format(TimeFormat)
	} else {
		callValues["fileCreatedAt"] = s.ModTime().UTC().Format(TimeFormat)
	}
	callValues["fileModifiedAt"] = s.ModTime().UTC().Format(TimeFormat)
	callValues["isFavorite"] = myBool(la.Favorite).String()
	callValues["fileExtension"] = ext
	callValues["duration"] = formatDuration(0)
	callValues["isReadOnly"] = "false"
	callValues["isArchived"] = myBool(la.Archived).String()
	return callValues
}

func (ic *ImmichClient) writeMultipartFields(m *multipart.Writer, callValues map[string]string) error {
	for key, value := range callValues {
		err := m.WriteField(key, value)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ic *ImmichClient) writeFilePart(m *multipart.Writer, f io.Reader, originalFileName, _ string) error {
	w, err := m.CreateFormFile("assetData", originalFileName)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, f)
	return err
}

func (ic *ImmichClient) writeSideCarPart(m *multipart.Writer, la *assets.Asset) error {
	scName := path.Base(la.OriginalFileName) + ".xmp"

	w, err := m.CreateFormFile("sidecarData", scName)
	if err != nil {
		return err
	}
	scf, err := la.FromSideCar.File.Open()
	if err != nil {
		return err
	}
	defer scf.Close()
	_, err = io.Copy(w, scf)
	return err
}
