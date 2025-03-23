# Use a Go image to compile the code
FROM golang:1.19

# Set the working directory inside the container
WORKDIR /go/src/app

# Copy your Go code and go.mod file into the container
COPY dingus-aid.go .
COPY go.mod .

CMD ["bash", "./make-binary.sh"]