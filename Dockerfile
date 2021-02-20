FROM golang:1.16

RUN apt update -y \
	&& apt install -y xvfb x11vnc \
	gstreamer1.0-x libgstreamer-plugins-base1.0-dev
