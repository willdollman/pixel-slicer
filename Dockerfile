FROM golang:latest

# Install dependencies
RUN apt update && apt install -y libvips-dev ffmpeg

# Copy files
WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

CMD [ "pixel-slicer" ]
