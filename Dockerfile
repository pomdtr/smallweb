FROM denoland/deno:1.46.3

# install curl
RUN apt-get update \
    && apt-get install -y curl \
    && rm -rf /var/lib/apt/lists /var/cache/apt/archives

# Use a non-root user for better security
RUN useradd -m smallweb

# Combine RUN commands to reduce layers and use curl instead of apt-get for installation
RUN curl -fsSL "https://install.smallweb.run?v=${SMALLWEB_VERSION:-0.14.5}&target_dir=/usr/local/bin" | sh \
    && chmod +x /usr/local/bin/smallweb

# Set environment variables
ENV SMALLWEB_DATA_DIR=/var/lib/smallweb \
    SMALLWEB_DIR=/smallweb \
    SMALLWEB_HOST=0.0.0.0 \
    SMALLWEB_PORT=7777

# Create necessary directories and set permissions
RUN mkdir -p "$SMALLWEB_DATA_DIR" "$SMALLWEB_DIR" \
    && chown -R smallweb:smallweb "$SMALLWEB_DATA_DIR" "$SMALLWEB_DIR"

# Switch to non-root user
USER smallweb

# Declare volumes
VOLUME ["$SMALLWEB_DATA_DIR", "$SMALLWEB_DIR"]

# Expose port
EXPOSE 7777

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/smallweb", "up"]
