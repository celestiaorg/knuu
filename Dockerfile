# website::tag::1:: Build a simple Docker image that contains a text file with the contents "Hello, World!"
FROM ubuntu:18.04

# Define a build argument with a default value
ARG MESSAGE="Hello, World!"

# Use the build argument in the RUN instruction
RUN echo "${MESSAGE}" > /test.txt

CMD ["sleep", "infinity"]
