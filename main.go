package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	"image/color"
	"image/draw"
	"image/jpeg"

	pigo "github.com/esimov/pigo/core"
	"github.com/morikuni/aec"
)

//go:embed cascade/facefinder
//go:embed cascade/puploc
var cascadeFolder embed.FS

var (
	buildTime string = "1970-01-01T00:00:00UTC"
	version   string = "0.0dev"
)

// Specifies a location with center point and size being side of quad (or diameter of circle)
type Location struct {
	CX   int `json:"cx"`
	CY   int `json:"cy"`
	Size int `json:"size"`
}

func NewLocationFromOffsetCenterSize(offsetX, offsetY, centerX, centerY, size int) *Location {
	loc := Location{
		CX:   offsetX + centerX,
		CY:   offsetY + centerY,
		Size: size,
	}
	return &loc
}

type FaceResult struct {
	Face     *Location `json:"face,omitempty"`
	LeftEye  *Location `json:"lefteye,omitempty"`
	RightEye *Location `json:"righteye,omitempty"`
	Quality  float32   `json:"quality"`
}

type FoundFaces struct {
	Faces []*FaceResult `json:"faces"`
}

func parseBoundingBox(bboxStr string) []float64 {
	bboxStr = strings.Trim(bboxStr, "'\"")
	bboxParams := strings.Split(bboxStr, ",")
	// Default params, whole image
	bounds := []float64{0.0, 0.0, 1.0, 1.0}
	if len(bboxParams) > 4 {
		bboxParams = bboxParams[:4]
	}
	for index, param := range bboxParams {
		param = strings.TrimSpace(param)
		val, err := strconv.ParseFloat(param, 64)
		if err == nil {
			bounds[index] = val
		}
	}
	return bounds
}

func getSubRectangleForImage(img image.Image, bboxParams []float64) *image.Rectangle {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	if len(bboxParams) < 4 {
		panic("bboxParams should have four values")
	}

	// bbox holds four values: x,y,w,h
	x1 := int(float64(width) * bboxParams[0])
	y1 := int(float64(height) * bboxParams[1])
	x2 := int(float64(width) * (bboxParams[0] + bboxParams[2]))
	y2 := int(float64(height) * (bboxParams[1] + bboxParams[3]))

	if x1 < 0 {
		x1 = 0
	}
	if y1 < 0 {
		y1 = 0
	}
	if x2 > width {
		x2 = width
	}
	if y2 > height {
		y2 = height
	}

	// Rect contains points [x1,x2) x [y1,y2)
	rect := image.Rect(x1, y1, x2, y2)
	return &rect
}

const figletStr = `
 _______                  ______         __              __   
|   _   .---.-.----.-----|   _  \ .-----|  |_.-----.----|  |_ 
|.  1___|  _  |  __|  -__|.  |   \|  -__|   _|  -__|  __|   _|
|.  __) |___._|____|_____|.  |    |_____|____|_____|____|____|
|:  |                    |:  1    /                           
|::.|                    |::.. . /                            
'---'                    '------'        									  										   
`

const usageStr = `Usage of facedetect:
facedetect [flags] -in image.jpg

This tool tries to detect front faces in the image file specified with the -in flag.
Supported image formats are: jpeg.

Required flags:
  -in string
 		Image file

Optional flags:
  -bbox string
		Bounding box to limit the search, in normalized coordinates. x,y,w,h (default "0,0,1,1")
  -debug
		Output a debug.jpg
  -fq float
		Min face quality to accept the face. (default 1)
  -fs float
		Min face size to accept the face. (default 40)
  -iou float
  		The intersection over union threshold for cluster detection (default 0.15)
  -pretty
		Pretty-print the json output
  -version
		Display version
  -h, --help
		Display this help

`

func printLogo() {
	logo := aec.LightRedF.Apply(figletStr)
	fmt.Println(logo)
}

func usage() {
	fmt.Fprint(os.Stderr, usageStr)
	os.Exit(2)
}

func main() {

	flag.Usage = usage
	debug := flag.Bool("debug", false, "Output a debug.jpg")
	imageFileName := flag.String("in", "", "Image file")
	qualityThreshold := flag.Float64("fq", 1.0, "Min face quality to accept the face.")
	sizeThreshold := flag.Float64("fs", 40, "Min face size to accept the face.")
	iouThreshold := flag.Float64("iou", 0.15, "The intersection over union threshold for cluster detection")
	bboxStr := flag.String("bbox", "0,0,1,1", "Bounding box to limit the search, in normalized coordinates. x,y,w,h")
	prettyJson := flag.Bool("pretty", false, "Pretty-print the json output")
	displayVersion := flag.Bool("version", false, "Display version")
	flag.Parse()

	if *displayVersion {
		printLogo()
		fmt.Printf("%s v%s (%s)\n", aec.LightRedF.Apply("FaceDetect"), aec.YellowF.Apply(version), aec.YellowF.Apply(buildTime))
		os.Exit(0)
	}

	if len(*imageFileName) == 0 {
		log.Fatal("Missing image file, use flag -in <image.jpg>. See usage help with -h.")
	}

	// bbox holds four values: x,y,w,h
	bboxParams := parseBoundingBox(*bboxStr)

	faceCascadeFile, err := cascadeFolder.ReadFile("cascade/facefinder")
	if err != nil {
		log.Fatalf("Error reading the facefinder cascade file: %v", err)
	}

	puplocCascadeFile, err := cascadeFolder.ReadFile("cascade/puploc")
	if err != nil {
		log.Fatalf("Error reading the puploc cascade file: %v", err)
	}

	src, err := pigo.GetImage(*imageFileName)
	if err != nil {
		log.Fatalf("Cannot open the image file: %v", err)
	}

	subRect := getSubRectangleForImage(src, bboxParams)

	// The left corner of the sub rectangle in the original image
	offsetX := subRect.Min.X
	offsetY := subRect.Min.Y

	newRect := image.Rect(0, 0, subRect.Dx(), subRect.Dy())
	newSrc := image.NewNRGBA(newRect)

	for y := subRect.Min.Y; y < subRect.Max.Y; y++ {
		for x := subRect.Min.X; x < subRect.Max.X; x++ {
			c := src.At(x, y)
			newSrc.Set(x-offsetX, y-offsetY, c)
		}
	}

	// Hmm this down't work since pigo.RgbToGrayscale doesn't work with subimages
	//newSrc = src.SubImage(subRect).(*image.NRGBA)

	pixels := pigo.RgbToGrayscale(newSrc)
	cols, rows := newSrc.Bounds().Dx(), newSrc.Bounds().Dy()

	sqBoxSize := int(math.Min(float64(newRect.Dx()), float64(newRect.Dy())))

	imgParams := &pigo.ImageParams{
		Pixels: pixels,
		Rows:   rows,
		Cols:   cols,
		Dim:    cols,
	}

	cParams := pigo.CascadeParams{
		MinSize:     20,
		MaxSize:     sqBoxSize,
		ShiftFactor: 0.1,
		ScaleFactor: 1.1,
		ImageParams: *imgParams,
	}

	pigoInst := pigo.NewPigo()

	faceClassifier, err := pigoInst.Unpack(faceCascadeFile)
	if err != nil {
		log.Fatalf("Error unpacking the facefinder cascade file: %s", err)
	}

	pupilLocator := pigo.NewPuplocCascade()
	pupilLocator, err = pupilLocator.UnpackCascade(puplocCascadeFile)
	if err != nil {
		log.Fatalf("Error unpacking the puploc cascade file: %s", err)
	}

	angle := 0.0 // cascade rotation angle. 0.0 is 0 radians and 1.0 is 2*pi radians

	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	dets := faceClassifier.RunCascade(cParams, angle)

	// Calculate the intersection over union (IoU) of two clusters.
	faces := faceClassifier.ClusterDetections(dets, *iouThreshold)

	qThresh := float32(*qualityThreshold)
	faceSizeThresh := int(*sizeThreshold)
	// Perturb is the number of random points to try when classifying pupils. Max is 63.
	perturb := 63

	foundFaces := make([]*FaceResult, 0)
	for _, face := range faces {

		if face.Q >= qThresh {
			if *debug {
				fill := &image.Uniform{color.NRGBA{R: 0, G: 255, B: 0, A: 64}}
				r := image.Rect(face.Col-face.Scale/2, face.Row-face.Scale/2, face.Col+face.Scale/2, face.Row+face.Scale/2)
				draw.Draw(newSrc, r, fill, image.Point{}, draw.Over)
			}

			if face.Scale >= faceSizeThresh {

				// Face of acceptable size
				faceResult := FaceResult{
					Face:    NewLocationFromOffsetCenterSize(offsetX, offsetY, face.Col, face.Row, face.Scale),
					Quality: face.Q,
				}

				// Detect left eye
				puploc := &pigo.Puploc{
					Row:      face.Row - int(0.075*float32(face.Scale)),
					Col:      face.Col - int(0.175*float32(face.Scale)),
					Scale:    float32(face.Scale) * 0.25,
					Perturbs: perturb,
				}
				leftEye := pupilLocator.RunDetector(*puploc, *imgParams, angle, false)
				if leftEye.Row > 0 && leftEye.Col > 0 {
					if *debug {
						fill := &image.Uniform{color.NRGBA{R: 255, G: 0, B: 0, A: 192}}
						r := image.Rect(leftEye.Col-int(leftEye.Scale/2), leftEye.Row-int(leftEye.Scale/2), leftEye.Col+int(leftEye.Scale/2), leftEye.Row+int(leftEye.Scale/2))
						draw.Draw(newSrc, r, fill, image.Point{}, draw.Over)
					}
					faceResult.LeftEye = NewLocationFromOffsetCenterSize(offsetX, offsetY, leftEye.Col, leftEye.Row, int(leftEye.Scale))
				}

				// Detect right eye
				puploc = &pigo.Puploc{
					Row:      face.Row - int(0.075*float32(face.Scale)),
					Col:      face.Col + int(0.185*float32(face.Scale)),
					Scale:    float32(face.Scale) * 0.25,
					Perturbs: perturb,
				}

				rightEye := pupilLocator.RunDetector(*puploc, *imgParams, angle, false)
				if rightEye.Row > 0 && rightEye.Col > 0 {
					if *debug {
						fill := &image.Uniform{color.NRGBA{R: 255, G: 0, B: 0, A: 192}}
						r := image.Rect(rightEye.Col-int(rightEye.Scale/2), rightEye.Row-int(rightEye.Scale/2), rightEye.Col+int(rightEye.Scale/2), rightEye.Row+int(rightEye.Scale/2))
						draw.Draw(newSrc, r, fill, image.Point{}, draw.Over)
					}
					faceResult.RightEye = NewLocationFromOffsetCenterSize(offsetX, offsetY, rightEye.Col, rightEye.Row, int(rightEye.Scale))
				}

				foundFaces = append(foundFaces, &faceResult)

			}
		}
	}

	if *debug {
		out, err := os.Create("./debug.jpg")
		if err != nil {
			log.Fatal(err)
		}

		opt := jpeg.Options{
			Quality: 95,
		}

		err = jpeg.Encode(out, newSrc, &opt)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Output json
	result := FoundFaces{
		Faces: foundFaces,
	}

	var (
		bytes []byte
	)
	if *prettyJson {
		bytes, err = json.MarshalIndent(result, "", "   ")
	} else {
		bytes, err = json.Marshal(result)
	}

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(bytes))
}
