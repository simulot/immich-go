package duplicateitem

type EditGroup struct {
	Group
	Index int // index in the full list
}

type BackFromGroup struct{}

type DisplayThumbnail string
