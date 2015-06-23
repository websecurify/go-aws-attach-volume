FROM scratch

# ---
# ---
# ---

COPY go-aws-attach-volume /

# ---
# ---
# ---

ENTRYPOINT ["/go-aws-attach-volume"]

# ---
