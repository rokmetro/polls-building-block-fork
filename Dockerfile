FROM public.ecr.aws/docker/library/golang:1.24-alpine as builder

ENV CGO_ENABLED=0

RUN mkdir /polls-app
WORKDIR /polls-app
# Copy the source from the current directory to the Working Directory inside the container
COPY . .
RUN make

FROM public.ecr.aws/docker/library/alpine:3.21.3

#we need timezone database + certificates
RUN apk add --no-cache tzdata ca-certificates

COPY --from=builder /polls-app/bin/polls /
COPY --from=builder /polls-app/driver/web/docs/gen/def.yaml /driver/web/docs/gen/def.yaml

COPY --from=builder /polls-app/driver/web/authorization_model.conf /driver/web/authorization_model.conf
COPY --from=builder /polls-app/driver/web/authorization_policy.csv /driver/web/authorization_policy.csv

COPY --from=builder /polls-app/vendor/github.com/rokwire/core-auth-library-go/v3/authorization/authorization_model_scope.conf /polls-app/vendor/github.com/rokwire/core-auth-library-go/v3/authorization/authorization_model_scope.conf
COPY --from=builder /polls-app/vendor/github.com/rokwire/core-auth-library-go/v3/authorization/authorization_model_string.conf /polls-app/vendor/github.com/rokwire/core-auth-library-go/v3/authorization/authorization_model_string.conf

#we need timezone database
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo 

ENTRYPOINT ["/polls"]
