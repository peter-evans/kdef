FROM scratch
COPY kdef /usr/bin/kdef
ENTRYPOINT ["/usr/bin/kdef"]
