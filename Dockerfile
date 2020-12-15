FROM alpine:edge AS mozjpeg
RUN echo "http://dl-cdn.alpinelinux.org/alpine/edge/testing" >> /etc/apk/repositories
RUN apk add --no-cache alpine-sdk sudo
RUN adduser -s /bin/sh -D -G abuild abuild
RUN echo "%abuild ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers
USER abuild
ENV ABUILD_LAST_COMMIT "1"
ENV SOURCE_DATE_EPOCH "1"
RUN abuild-keygen -a -i -n -q

# Buildind mozjpeg and installing it
WORKDIR /tmp/mozjpeg
COPY --chown=abuild:abuild alpine/mozjpeg/APKBUILD ./
RUN sudo chown abuild:abuild ./
RUN abuild deps && abuild -r
RUN sudo apk add --allow-untrusted /home/abuild/packages/tmp/x86_64/*.apk

# Building tiff and installing it
WORKDIR /tmp/tiff
COPY --chown=abuild:abuild alpine/tiff/* ./
RUN sudo chown abuild:abuild ./
RUN abuild deps && abuild -r
RUN sudo apk add --allow-untrusted /home/abuild/packages/tmp/x86_64/*.apk

# Building lcms2 and installing it
WORKDIR /tmp/lcms2
COPY --chown=abuild:abuild alpine/lcms2/* ./
RUN sudo chown abuild:abuild ./
RUN abuild deps && abuild -r
RUN sudo apk add --allow-untrusted /home/abuild/packages/tmp/x86_64/*.apk

# Building libimagequant and installing it
WORKDIR /tmp/libimagequant
COPY --chown=abuild:abuild alpine/libimagequant/APKBUILD ./
RUN sudo chown abuild:abuild ./
RUN abuild deps && abuild -r
RUN sudo apk add --allow-untrusted /home/abuild/packages/tmp/x86_64/*.apk

# Building libvips
WORKDIR /tmp/vips
COPY --chown=abuild:abuild alpine/vips/APKBUILD ./
RUN sudo chown abuild:abuild ./
RUN abuild deps && abuild -r


# Building reImage
FROM alpine:edge AS go
WORKDIR /go
RUN echo "http://dl-cdn.alpinelinux.org/alpine/edge/testing" >> /etc/apk/repositories
COPY --from=mozjpeg /home/abuild/packages/tmp/x86_64/*.apk /tmp/
RUN apk add --allow-untrusted /tmp/*.apk && apk add --no-cache go git fftw-dev musl-dev
COPY *.go go.sum go.mod ./
RUN go mod vendor
RUN go build -o reImage *.go


# Create Release image without dev dependencies
FROM alpine:edge AS release
WORKDIR /usr/local/bin/
RUN echo "http://dl-cdn.alpinelinux.org/alpine/edge/testing" >> /etc/apk/repositories
COPY --from=mozjpeg /home/abuild/packages/tmp/x86_64/mozjpeg*.apk /tmp/
COPY --from=mozjpeg /home/abuild/packages/tmp/x86_64/vips*.apk /tmp/
COPY --from=mozjpeg /home/abuild/packages/tmp/x86_64/lcms2*.apk /tmp/
COPY --from=mozjpeg /home/abuild/packages/tmp/x86_64/tiff*.apk /tmp/
COPY --from=mozjpeg /home/abuild/packages/tmp/x86_64/libimagequant*.apk /tmp/
RUN apk add --allow-untrusted /tmp/*.apk && apk add --no-cache ca-certificates
COPY --from=go /go/reImage .
ENV CFG_LISTEN ":7075"
CMD ["./reImage"]
