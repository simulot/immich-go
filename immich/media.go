package immich

type UnsupportedMedia struct {
	msg string
}

func (u UnsupportedMedia) Error() string {
	return u.msg
}

func (u UnsupportedMedia) Is(target error) bool {
	_, ok := target.(*UnsupportedMedia)
	return ok
}

func (ic *ImmichClient) TypeFromExt(ext string) string {
	return ic.supportedMediaTypes.TypeFromExt(ext)
}

func (ic *ImmichClient) IsExtensionPrefix(ext string) bool {
	return ic.supportedMediaTypes.IsExtensionPrefix(ext)
}

func (ic *ImmichClient) IsIgnoredExt(ext string) bool {
	return ic.supportedMediaTypes.IsIgnoredExt(ext)
}
