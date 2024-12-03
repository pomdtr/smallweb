FROM denoland/deno:2.1.1

# install curl
RUN apt-get update \
    && apt-get install -y curl \
    && rm -rf /var/lib/apt/lists /var/cache/apt/archives

# Use a non-root user for better security
RUN useradd --create-home --user-group --shell $(which bash) smallweb

ARG SMALLWEB_VERSION=0.17.10

# Combine RUN commands to reduce layers and use curl instead of apt-get for installation
RUN curl -fsSL "https://install.smallweb.run?v=${SMALLWEB_VERSION}&target_dir=/usr/local/bin" | sh \
    && chmod +x /usr/local/bin/smallweb

# Switch to non-root user
USER smallweb
WORKDIR /home/smallweb

# Set environment variables
ENV HOME=/home/smallweb \
    SMALLWEB_DIR=/home/smallweb/smallweb \
    SMALLWEB_ADDR=0.0.0.0:7777

VOLUME ["$SMALLWEB_DIR"]

# Expose port
EXPOSE 7777

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/smallweb", "up"]
