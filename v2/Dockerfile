FROM golang:1.13

WORKDIR /wobbotfet

ADD . /wobbotfet

ENV PORT 4080
EXPOSE $PORT

RUN cd /wobbotfet && go build

ENTRYPOINT ["./wobbotfet"]
