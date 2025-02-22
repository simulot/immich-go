//go:build e2e
// +build e2e

package upload_test

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"testing"
	"time"

	"github.com/simulot/immich-go/app/cmd"
	"github.com/simulot/immich-go/internal/e2eTests/e2e"
	"golang.org/x/exp/rand"
)

// TestUploadBurstInAlbums show the immich problem when a stack is included into an album
// immich version v1.123.0
func TestUploadBurstInAlbums(t *testing.T) {
	e2e.InitMyEnv()
	e2e.ResetImmich(t)

	ctx := context.Background()
	c, a := cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-folder",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--no-ui",
		"--into-album=ALBUM",
		"--log-level=debug",
		"--manage-raw-jpeg=KeepRaw",
		"--manage-burst=stack",
		e2e.MyEnv("IMMICHGO_TESTFILES") + "/burst/storm",
	})

	// let's start
	err := c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}
}

func TestUploadBurstFromZip(t *testing.T) {
	e2e.InitMyEnv()
	e2e.ResetImmich(t)

	ctx := context.Background()
	c, a := cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-folder",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--no-ui",
		"--into-album=ALBUM",
		"--manage-raw-jpeg=KeepRaw",
		"--manage-burst=stack",
		"--log-level=DEBUG",
		e2e.MyEnv("IMMICHGO_TESTFILES") + "/burst/storm.zip",
	})

	// let's start
	err := c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}
}

func TestUploadArchive(t *testing.T) {
	e2e.InitMyEnv()
	e2e.ResetImmich(t)
	ctx := context.Background()

	c, a := cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-folder",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--no-ui",
		e2e.MyEnv("IMMICHGO_TESTFILES") + "/testArchive",
	})

	// let's start
	err := c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}
}

func TestUploadFromFastFotoFolder(t *testing.T) {
	e2e.InitMyEnv()
	e2e.ResetImmich(t)
	tmp, list, cleanup := create_test_epsonfoto_folder(t, 10, 5)
	defer cleanup()

	ctx := context.Background()

	c, a := cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-folder",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--no-ui",
		"--log-level=DEBUG",
		"--manage-epson-fastfoto",
		tmp,
	})

	// let's start
	err := c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}

	_ = tmp
	_ = list
}

func TestUpgradeFromFolder(t *testing.T) {
	e2e.InitMyEnv()
	e2e.ResetImmich(t)
	client, err := e2e.GetImmichClient()
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	c, a := cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-folder",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--log-level=debug",
		"--date-from-name",
		// "--no-ui",
		"../../../app/cmd/upload/TEST_DATA/folder/low",
	})

	// let's start
	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}
	e2e.WaitingForJobsEnding(ctx, client, t)

	c, a = cmd.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"upload", "from-folder",
		"--server=" + e2e.MyEnv("IMMICHGO_SERVER"),
		"--api-key=" + e2e.MyEnv("IMMICHGO_APIKEY"),
		"--date-from-name",
		"--api-trace",
		"--log-level=debug",
		// "--no-ui",
		"../../../app/cmd/upload/TEST_DATA/folder/high",
	})

	// let's start
	err = c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}
}

var forenames = []string{
	"James", "Mary", "John", "Patricia", "Robert", "Jennifer", "Michael", "Linda", "William", "Elizabeth",
	"David", "Barbara", "Richard", "Susan", "Joseph", "Jessica", "Thomas", "Sarah", "Charles", "Karen",
	"Christopher", "Nancy", "Daniel", "Lisa", "Matthew", "Betty", "Anthony", "Margaret", "Mark", "Sandra",
	"Donald", "Ashley", "Steven", "Kimberly", "Paul", "Emily", "Andrew", "Donna", "Joshua", "Michelle",
	"Kenneth", "Dorothy", "Kevin", "Carol", "Brian", "Amanda", "George", "Melissa", "Edward", "Deborah",
}

var cityNames = []string{
	"Paris", "London", "NewYork", "Tokyo", "Berlin", "Madrid", "Rome", "Beijing", "Moscow", "Sydney",
	"Toronto", "Dubai", "Singapore", "HongKong", "LosAngeles", "Chicago", "Boston", "SanFrancisco", "Miami", "Dallas",
	"Seattle", "Vancouver", "Melbourne", "Delhi", "Bangkok", "Istanbul", "Cairo", "MexicoCity", "BuenosAires", "SaoPaulo",
	"Jakarta", "Mumbai", "Shanghai", "Seoul", "KualaLumpur", "Manila", "Lisbon", "Vienna", "Zurich", "Stockholm",
	"Helsinki", "Oslo", "Copenhagen", "Brussels", "Amsterdam", "Warsaw", "Prague", "Budapest", "Athens", "Dublin",
}

// Create a test folder and return the cleanup function
func create_test_folder(t *testing.T, folders, filesPerFolder int) (string, map[string][]string, func()) {
	rand.Seed(uint64(time.Now().UnixNano()))

	dirList := map[string][]string{}
	tmpDir := os.TempDir()
	tmpDir, err := os.MkdirTemp(tmpDir, "upload_test_folder")
	if err != nil {
		t.Fatal(err)
	}

	for f := 0; f < folders; f++ {
		list := []string{}
		city := cityNames[rand.Intn(len(cityNames))]
		for {
			if _, ok := dirList[city]; !ok {
				break
			}
			city = cityNames[rand.Intn(len(cityNames))]
		}
		folderName := city
		folderPath := tmpDir + "/" + folderName
		for i := 0; i < filesPerFolder; i++ {
			person := forenames[rand.Intn(len(forenames))]

			fileName := fmt.Sprintf("%s_%d.jpg", person, i+1)
			filePath := folderPath + "/" + fileName

			err := os.MkdirAll(folderPath, 0o755)
			if err != nil {
				t.Fatal(err)
			}

			err = generateRandomImage(filePath)
			if err != nil {
				t.Fatal(err)
			}
			list = append(list, filePath)
		}
		dirList[folderName] = list
	}
	return tmpDir, dirList, func() {
		os.RemoveAll(tmpDir)
	}
}

// Create a test folder and return the cleanup function
func create_test_epsonfoto_folder(t *testing.T, folders, filesPerFolder int) (string, map[string][]string, func()) {
	rand.Seed(uint64(time.Now().UnixNano()))

	dirList := map[string][]string{}
	tmpDir := os.TempDir()
	tmpDir, err := os.MkdirTemp(tmpDir, "test_epsonfoto_folder")
	if err != nil {
		t.Fatal(err)
	}

	for f := 0; f < folders; f++ {
		list := []string{}
		city := cityNames[rand.Intn(len(cityNames))]
		for {
			if _, ok := dirList[city]; !ok {
				break
			}
			city = cityNames[rand.Intn(len(cityNames))]
		}
		for i := 0; i < filesPerFolder; i++ {
			base := fmt.Sprintf("%s_%04d", city, i+1)

			for _, suffix := range []string{".jpg", "_a.jpg", "_b.jpg"} {
				name := tmpDir + "/" + base + suffix
				err = generateRandomImage(name)
				if err != nil {
					t.Fatal(err)
				}
				list = append(list, name)
			}
		}
		dirList[city] = list
	}
	return tmpDir, dirList, func() {
		os.RemoveAll(tmpDir)
	}
}

func generateRandomImage(path string) error {
	width := 32
	height := 32

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(rand.Intn(256)),
				G: uint8(rand.Intn(256)),
				B: uint8(rand.Intn(256)),
				A: 255,
			})
		}
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	err = jpeg.Encode(file, img, nil)
	if err != nil {
		return err
	}

	return nil
}
