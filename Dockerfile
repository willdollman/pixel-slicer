# Stage 1: Build Go app
FROM golang:latest

RUN apt update && apt install -y libvips-dev

# Copy files
WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...


# Stage 2: Install media encoder dependencies and copy app binary
# Using ubuntu rather than alpine due to issues with CGO and libvips
FROM ubuntu:latest

# Install dependencies
RUN apt update \
	&& DEBIAN_FRONTEND=noninteractive apt install -y libvips ffmpeg \
	&& rm -rf /var/lib/apt/lists/*

WORKDIR /root/
COPY --from=0 /go/bin/pixel-slicer ./

CMD [ "./pixel-slicer" ]
