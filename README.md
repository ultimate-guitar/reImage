# reImage

## Features
1. No os.exec call of pngquant or mozjpeg. Only bindings to libraries.
2. Using libvips for fast image resizing
3. Docker container uses libvips compiled with mozjpeg instead of libjpeg-turbo. MozJPEG makes tradeoffs that are intended to benefit Web use cases and focuses solely on improving encoding, so it's best used as part of a Web encoding workflow.
4. png images are optimized with libimagequant (backend library of pngquant)


## Install steps
1. Deploy it with Docker
2. Configure frontend
3. Enjoy!

## Deploy
1. Pull docker container:  `docker pull larrabee/reimage`
2. Run it with `docker run -d -p 7075:7075 reimage` or use docker-compose config:
```
version: '2.2'
services:
  reImage:
      image: larrabee/reimage
      restart: always
      network: host # Use host nework for better performance
      scale: 8  # Replace it with your vCPU count
      environment:
        CFG_LISTEN: 127.0.0.1:7075
        GOMAXPROCS: 1  # Set 1 for low latency, 2-4 for max throughput
```
3. Why I should run multiple workers?
   See [issue #1](/../../issues/1)

## Configure frontend
This is basic Nginx config for online resizing with nginx cache.
```
events {
}

http {
    proxy_cache_path /var/www/resize_cache levels=1:2 
    keys_zone=resized_img:64m inactive=48h max_size=5G use_temp_path=off;

    server {
        listen       80;
        server_name  example.com;
        root /var/www/example.com/;

        location /img/ {
           location ~* \.(jpg|jpeg|png|ico)(\@[0-9xX]+)$ {
             proxy_pass http://localhost:7075; #Node with running image resizer
             proxy_set_header X-RESIZE-SCHEME "http";
             proxy_set_header X-RESIZE-BASE "example.com";
             #Nginx cache. Optional
             proxy_cache_valid  200 48h;
             proxy_cache_valid  404 400 5m;
             proxy_cache_valid 500 502 503 504 10s;
             add_header X-Cache-Status $upstream_cache_status;
             proxy_cache resized_img;
           }
        }
    }
}
```
Required options:
1. Replace `localhost:7075` with your hostname and port if you ran resizer in different node.
2. Set `X-RESIZE-SCHEME "http"` if your image server with original images run over http (by default use https). 
3. Set you image server hostname here: `X-RESIZE-BASE "example.com"`

Optional:
4. Set `X-RESIZE-QUALITY` header if you want to override quality settings. Default value: 80, allowed values: 1-100
5. Set `X-RESIZE-COMPRESSION` header if you want to override compression settings. Default value: 6, allowed values: 0-9

## Enjoy!
* Upload [test image](http://www.publicdomainpictures.net/pictures/110000/velka/green-mountain-valley.jpg) to `/img/test.jpg` on your server
* Get original image [http://example.com/img/test.jpg](http://example.com/img/test.jpg) (naturally it must be present on your server).
* Get resized to 1280x720 version [http://example.com/img/test.jpg@1280](http://example.com/img/test.jpg@1280) (height resolution will be calculate automatically)
* Another way: [http://example.com/img/test.jpg@x720](http://example.com/img/test.jpg@x720) (width resolution will be calculate automatically)
* Get resized to 500x500 version (image will be resized and striped to 500x500) [http://example.com/img/test.jpg@500x500](http://example.com/img/test.jpg@500x500)

## Benchmark
See [benchmark Wiki page](https://github.com/ultimate-guitar/reImage/wiki/Benchmark)

## LICENSE
MIT
