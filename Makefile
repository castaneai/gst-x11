build:
	docker build -t gst-x11 .

bash: build
	docker run --rm -it -v $(realpath .):/app -p 5900:5900 gst-x11 bash
