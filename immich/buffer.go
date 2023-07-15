package immich

/*
var bytesPool = sync.Pool{
	New: func() any {
		return []byte{}
	},
}

type Buffer struct {
	b   []byte
	buf *bytes.Buffer
}

func NewBuffer() *Buffer {
	b := Buffer{}
	b.b = bytesPool.Get().([]byte)
	b.buf = bytes.NewBuffer(b.b)
	return &b
}

func (buf *Buffer) Read(b []byte) (int, error) {
	return buf.buf.Read(b)
}

func (buf *Buffer) Write(b []byte) (int, error) {
	return buf.buf.Write(b)
}

func (b *Buffer) Close() error {
	b.buf = nil
	bytesPool.Put(b.b)
	return nil
}
*/
/*
const buffersNumber = 20

var buffers = make(chan any, buffersNumber)

func initBuffers() {
	for i := 0; i < buffersNumber; i++ {
		buffers <- nil
	}
}

type Buffer struct {
	*bytes.Buffer
}

func NewBuffer() *Buffer {
	<-buffers
	b := bytes.NewBuffer(nil)
	return &Buffer{
		Buffer: b,
	}
}

func (b *Buffer) Close() error {
	buffers <- nil
	return nil
}
*/
