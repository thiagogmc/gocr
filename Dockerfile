FROM golang:latest

RUN apt-get update -qq

# You need librariy files and headers of tesseract and leptonica.
# When you miss these or LD_LIBRARY_PATH is not set to them,
# you would face an error: "tesseract/baseapi.h: No such file or directory"
RUN apt-get install -y -qq libtesseract-dev libleptonica-dev

# In case you face TESSDATA_PREFIX error, you minght need to set env vars
# to specify the directory where "tessdata" is located.
ENV TESSDATA_PREFIX=/usr/share/tesseract-ocr/4.00/tessdata/

# Load languages.
# These {lang}.traineddata would b located under ${TESSDATA_PREFIX}/tessdata.
RUN apt-get install -y -qq \
  tesseract-ocr-eng \
  tesseract-ocr-por

RUN apt-get install -y git
RUN apt-get install -y libjpeg62-turbo-dev

# Set the Current Working Directory inside the container
WORKDIR /app/gocr

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

# Build the Go app
RUN go build -o ./out/gocr .

# This container exposes port 8080 to the outside world
EXPOSE 3000

# Run the binary program produced by `go install`
CMD ["./out/gocr"]