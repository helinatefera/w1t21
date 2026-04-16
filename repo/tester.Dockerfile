FROM node:20-alpine
RUN apk add --no-cache bash curl python3 coreutils
WORKDIR /workspace
ENTRYPOINT ["sh", "-lc"]
