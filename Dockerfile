FROM scratch

ADD bin/cargo_docker /bin/cargo
ENV PATH="/bin"

ENTRYPOINT ["cargo"]
CMD ["cargo"]
