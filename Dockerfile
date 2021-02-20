FROM ghcr.io/castaneai/wine:6.0-groovy

ENV DEBIAN_FRONTEND noninteractive
RUN apt update -y \
	&& apt install -y wget tar gcc xvfb x11vnc x11-apps
ENV PATH $PATH:/usr/local/go/bin
RUN wget https://golang.org/dl/go1.16.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go1.16.linux-amd64.tar.gz

RUN apt install -y libgstreamer1.0-0 \
    libgstreamer1.0-dev \
    gstreamer1.0-plugins-base libgstreamer-plugins-base1.0-dev \
    gstreamer1.0-plugins-good \
    gstreamer1.0-plugins-bad \
    gstreamer1.0-plugins-ugly \
    gstreamer1.0-x

WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .

ENV WINEDLLOVERRIDES "mscoree=d;mshtml=d"
COPY entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]