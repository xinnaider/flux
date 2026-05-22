# Build stage
FROM node:22-alpine AS build

WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

# Final stage — node serves static files
FROM node:22-alpine

RUN npm i -g serve
COPY --from=build /app/dist /app/dist
EXPOSE 80

CMD ["serve", "/app/dist", "-l", "80"]
