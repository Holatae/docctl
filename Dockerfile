FROM fedora:44
LABEL authors="victor"

# Installera GPG, Pandoc och wget (för att ladda ner Typst)
RUN dnf copr enable claaj/typst -y

RUN dnf5 update && dnf5 install -y gnupg wget pandoc go typst






# Sätt din arbetskatalog
# Skapa en mapp för tester med fulla rättigheter (så slipper vi /tmp-bråket)
# RUN mkdir -p /app/.gotmp && chmod 777 /app/.gotmp
# ENV GOTMPDIR=/app/.gotmp
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

CMD ["go", "test", "./...", "-v"]


# ENTRYPOINT ["top", "-b"]
