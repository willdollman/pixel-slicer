FROM golang:latest

# Install libvips-dev (dev unnecessary?) and ffmpeg
RUN apt update && apt install -y libvips-dev ffmpeg

WORKDIR /pixel-slicer
COPY cmd/ internal/ go.mod go.sum /pixel-slicer/

RUN go build /pixel-slicer/cmd/pixel-slicer/main.go

CMD [ "/pixel-slicer/" ]
