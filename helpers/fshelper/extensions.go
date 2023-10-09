package fshelper

import (
	"fmt"
	"strings"
)

// List from immich code:
// https://github.com/immich-app/immich/blob/8d5bf933601a3f2787a78c40e4c11862b96566e0/server/src/domain/domain.constant.ts#L26C17-L89C3

var supportedExtensions = map[string][]string{
	".3fr":  {"image/3fr", "image/x-hasselblad-3fr"},
	".ari":  {"image/ari", "image/x-arriflex-ari"},
	".arw":  {"image/arw", "image/x-sony-arw"},
	".avif": {"image/avif"},
	".cap":  {"image/cap", "image/x-phaseone-cap"},
	".cin":  {"image/cin", "image/x-phantom-cin"},
	".cr2":  {"image/cr2", "image/x-canon-cr2"},
	".cr3":  {"image/cr3", "image/x-canon-cr3"},
	".crw":  {"image/crw", "image/x-canon-crw"},
	".dcr":  {"image/dcr", "image/x-kodak-dcr"},
	".dng":  {"image/dng", "image/x-adobe-dng"},
	".erf":  {"image/erf", "image/x-epson-erf"},
	".fff":  {"image/fff", "image/x-hasselblad-fff"},
	".gif":  {"image/gif"},
	".heic": {"image/heic"},
	".heif": {"image/heif"},
	".iiq":  {"image/iiq", "image/x-phaseone-iiq"},
	".insp": {"image/jpeg"},
	".jpeg": {"image/jpeg"},
	".jpg":  {"image/jpeg"},
	".jxl":  {"image/jxl"},
	".k25":  {"image/k25", "image/x-kodak-k25"},
	".kdc":  {"image/kdc", "image/x-kodak-kdc"},
	".mrw":  {"image/mrw", "image/x-minolta-mrw"},
	".nef":  {"image/nef", "image/x-nikon-nef"},
	".orf":  {"image/orf", "image/x-olympus-orf"},
	".ori":  {"image/ori", "image/x-olympus-ori"},
	".pef":  {"image/pef", "image/x-pentax-pef"},
	".png":  {"image/png"},
	".psd":  {"image/psd", "image/vnd.adobe.photoshop"},
	".raf":  {"image/raf", "image/x-fuji-raf"},
	".raw":  {"image/raw", "image/x-panasonic-raw"},
	".rwl":  {"image/rwl", "image/x-leica-rwl"},
	".sr2":  {"image/sr2", "image/x-sony-sr2"},
	".srf":  {"image/srf", "image/x-sony-srf"},
	".srw":  {"image/srw", "image/x-samsung-srw"},
	".tif":  {"image/tiff"},
	".tiff": {"image/tiff"},
	".webp": {"image/webp"},
	".x3f":  {"image/x3f", "image/x-sigma-x3f"},

	".3gp":  {"video/3gpp"},
	".avi":  {"video/avi", "video/msvideo", "video/vnd.avi", "video/x-msvideo"},
	".flv":  {"video/x-flv"},
	".insv": {"video/mp4"},
	".m2ts": {"video/mp2t"},
	".m4v":  {"video/x-m4v"},
	".mkv":  {"video/x-matroska"},
	".mov":  {"video/quicktime"},
	".mp4":  {"video/mp4"},
	".mpg":  {"video/mpeg"},
	".mts":  {"video/mp2t"},
	".webm": {"video/webm"},
	".wmv":  {"video/x-ms-wmv"},
}

// MimeFromExt return the mime type of the extension. Return an error is the extension is not handled by the server.
func MimeFromExt(ext string) ([]string, error) {
	ext = strings.ToLower(ext)
	if l, ok := supportedExtensions[ext]; ok {
		return l, nil
	}
	return nil, fmt.Errorf("unsupported extension %s", ext)
}
