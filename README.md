
# FaceDetect 

FaceDetect is a simple cmd line tool to detect front faces in images and output found face locations as json. With front faces it is meant that both eyes are visible. 

## Go

To build it requires go. It has been tested with go 1.21. Perform **go mod tidy** to get dependencies. It uses the pigo face detect library.


## Usage
```
Usage of facedetect:
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

```


## Example

Detect faces in a screenshot:
![Screenshot from the movie Knives Out](docs/image1.jpg)

```sh
./facedetect.darwin.arm64 -in docs/image1.jpg -debug -fq 4.0 -fs 40 -pretty 
```
Detecting three faces in an image with specified min quality and size. The output json could look like:
```json
{
   "faces": [
      {
         "face": {
            "cx": 338,
            "cy": 151,
            "size": 127
         },
         "lefteye": {
            "cx": 315,
            "cy": 138,
            "size": 10
         },
         "righteye": {
            "cx": 362,
            "cy": 145,
            "size": 10
         },
         "quality": 69.340576
      },
      {
         "face": {
            "cx": 719,
            "cy": 204,
            "size": 42
         },
         "lefteye": {
            "cx": 714,
            "cy": 201,
            "size": 3
         },
         "righteye": {
            "cx": 727,
            "cy": 202,
            "size": 3
         },
         "quality": 75.25096
      },
      {
         "face": {
            "cx": 545,
            "cy": 182,
            "size": 46
         },
         "lefteye": {
            "cx": 540,
            "cy": 179,
            "size": 3
         },
         "righteye": {
            "cx": 552,
            "cy": 179,
            "size": 3
         },
         "quality": 22.660633
      }
   ]
}
```

The found faces (and eyes) are specified by a circle with center point (cx, cy) and diameter given by size. The value of quality specifies how good and the higher the better.

The generated debug.jpg image

![Found faces](docs/image1_faces.jpg)
