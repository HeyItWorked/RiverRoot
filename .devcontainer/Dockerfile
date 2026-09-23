FROM mcr.microsoft.com/devcontainers/go:1.26-bookworm

# Remove broken Yarn repo that causes apt-get update to fail during feature installs
RUN rm -f /etc/apt/sources.list.d/yarn.list
