FROM golang:1.22-alpine as builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./main.go
COPY cmd ./cmd/
COPY templates/ ./templates/
COPY worker ./worker/

RUN go build

FROM denoland/deno:1.45.2

COPY --from=builder /build/smallweb /usr/local/bin/smallweb
RUN apt-get update \
    && apt-get install -y sudo openssh-server \
    && cp /etc/ssh/sshd_config /etc/ssh/sshd_config-original \
    && sed -i 's/^#\s*Port.*/Port 2222/' /etc/ssh/sshd_config \
    && sed -i 's/^#\s*PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config \
    && mkdir -p /root/.ssh \
    && chmod 700 /root/.ssh \
    && mkdir /var/run/sshd \
    && chmod 755 /var/run/sshd \
    && rm -rf /var/lib/apt/lists /var/cache/apt/archives

ENV USERNAME "fly"
RUN useradd -m -s /bin/bash ${USERNAME}
RUN chown ${USERNAME}:${USERNAME} /home/${USERNAME}
RUN echo "%${USERNAME} ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers

WORKDIR /home/${USERNAME}
USER ${USERNAME}
COPY --chown=${USERNAME}:${USERNAME} entrypoint.sh /usr/local/bin/entrypoint.sh

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["smallweb", "up", "--host=0.0.0.0"]
