FROM ubuntu:latest

ADD pod-crash-controller pod-crash-controller
RUN chmod a+x ./pod-crash-controller
RUN apt-get update && apt-get install -y ca-certificates
RUN update-ca-certificates
ENTRYPOINT ["/bin/bash","-c","/pod-crash-controller"]