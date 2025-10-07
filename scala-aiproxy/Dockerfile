FROM hseeberger/scala-sbt:11.0.16.1_1.8.2_3.3.1 AS build

WORKDIR /app
COPY build.sbt .
COPY project ./project

# Download dependencies
RUN sbt update

# Copy source and build
COPY src ./src
RUN sbt assembly

FROM eclipse-temurin:21-jre-alpine

WORKDIR /app
COPY --from=build /app/target/scala-3.3.1/*-assembly-*.jar app.jar

EXPOSE 3000

ENV JAVA_OPTS="-Xmx512m -Xms256m"

ENTRYPOINT ["sh", "-c", "java $JAVA_OPTS -jar app.jar"]
