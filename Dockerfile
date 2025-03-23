# Use a Go image to compile the code
FROM golang:1.19

# Set the working directory inside the container
WORKDIR /go/src

# Copy code into the container
COPY app/ app/

CMD ["bash", "make-binary.sh"]