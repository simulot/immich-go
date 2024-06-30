package metadata

import (
	"testing"
	"time"
)

func TestMetadata_String(t *testing.T) {
	type fields struct {
		Description string
		DateTaken   time.Time
		Latitude    float64
		Longitude   float64
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "TimeOnly",
			fields: fields{
				DateTaken: time.Date(2000, 1, 2, 15, 32, 59, 0, time.UTC),
			},
			want: `<?xpacket begin='?' id='W5M0MpCehiHzreSzNTczkc9d'?>
<x:xmpmeta xmlns:x='adobe:ns:meta/' x:xmptk='Image::ExifTool 12.40'>
<rdf:RDF xmlns:rdf='http://www.w3.org/1999/02/22-rdf-syntax-ns#'>
 <rdf:Description rdf:about=''
  xmlns:exif='http://ns.adobe.com/exif/1.0/'>
  <exif:ExifVersion>0220</exif:ExifVersion>  <exif:DateTimeOriginal>2000-01-02T15:32:59Z</exif:DateTimeOriginal>
  <exif:GPSVersionID>2.3.0.0</exif:GPSVersionID>
 </rdf:Description>
</rdf:RDF>
</x:xmpmeta>
<?xpacket end='w'?>`,
		},
		{
			name: "DescriptionOnly",
			fields: fields{
				Description: "That's a < description > !",
			},
			want: `<?xpacket begin='?' id='W5M0MpCehiHzreSzNTczkc9d'?>
<x:xmpmeta xmlns:x='adobe:ns:meta/' x:xmptk='Image::ExifTool 12.40'>
<rdf:RDF xmlns:rdf='http://www.w3.org/1999/02/22-rdf-syntax-ns#'>
 <rdf:Description rdf:about=''
  xmlns:dc='http://purl.org/dc/elements/1.1/'>
  <dc:description>
   <rdf:Alt>
    <rdf:li xml:lang='x-default'>That&#39;s a &lt; description &gt; !</rdf:li>
   </rdf:Alt>
  </dc:description>
 </rdf:Description>
</rdf:RDF>
</x:xmpmeta>
<?xpacket end='w'?>`,
		},
		{
			name: "GPSOnly",
			fields: fields{
				Latitude:  71.1652089,
				Longitude: 25.7909877,
			},
			want: `<?xpacket begin='?' id='W5M0MpCehiHzreSzNTczkc9d'?>
<x:xmpmeta xmlns:x='adobe:ns:meta/' x:xmptk='Image::ExifTool 12.40'>
<rdf:RDF xmlns:rdf='http://www.w3.org/1999/02/22-rdf-syntax-ns#'>
 <rdf:Description rdf:about=''
  xmlns:exif='http://ns.adobe.com/exif/1.0/'>
  <exif:ExifVersion>0220</exif:ExifVersion>  <exif:GPSLatitude>71.165209</exif:GPSLatitude>
  <exif:GPSLongitude>25.790988</exif:GPSLongitude>
  <exif:GPSVersionID>2.3.0.0</exif:GPSVersionID>
 </rdf:Description>
</rdf:RDF>
</x:xmpmeta>
<?xpacket end='w'?>`,
		},
		{
			name: "All",
			fields: fields{
				Description: `That /!\ strange & dark <place> â ø`,
				DateTaken:   time.Date(2000, 1, 2, 15, 32, 59, 0, time.UTC),
				Latitude:    71.1652089,
				Longitude:   25.7909877,
			},
			want: `<?xpacket begin='?' id='W5M0MpCehiHzreSzNTczkc9d'?>
<x:xmpmeta xmlns:x='adobe:ns:meta/' x:xmptk='Image::ExifTool 12.40'>
<rdf:RDF xmlns:rdf='http://www.w3.org/1999/02/22-rdf-syntax-ns#'>
 <rdf:Description rdf:about=''
  xmlns:dc='http://purl.org/dc/elements/1.1/'>
  <dc:description>
   <rdf:Alt>
    <rdf:li xml:lang='x-default'>That /!\ strange &amp; dark &lt;place&gt; â ø</rdf:li>
   </rdf:Alt>
  </dc:description>
 </rdf:Description>
 <rdf:Description rdf:about=''
  xmlns:exif='http://ns.adobe.com/exif/1.0/'>
  <exif:ExifVersion>0220</exif:ExifVersion>  <exif:DateTimeOriginal>2000-01-02T15:32:59Z</exif:DateTimeOriginal>
  <exif:GPSLatitude>71.165209</exif:GPSLatitude>
  <exif:GPSLongitude>25.790988</exif:GPSLongitude>
  <exif:GPSVersionID>2.3.0.0</exif:GPSVersionID>
 </rdf:Description>
</rdf:RDF>
</x:xmpmeta>
<?xpacket end='w'?>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Metadata{
				Description: tt.fields.Description,
				DateTaken:   tt.fields.DateTaken,
				Latitude:    tt.fields.Latitude,
				Longitude:   tt.fields.Longitude,
			}
			if got := m.String(); got != tt.want {
				t.Errorf("Meta.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
