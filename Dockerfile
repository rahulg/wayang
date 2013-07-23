FROM rahulg/arch:july2013
MAINTAINER Rahul AG <r@hul.ag>

RUN pacman -Syu --noconfirm

RUN mkdir -p /opt/deploy

EXPOSE 8000
ENTRYPOINT ["/opt/deploy/wayang"]
CMD ["-conf", "/opt/deploy/config.json"]
