package main

import (
	"log"

	"github.com/gofiber/fiber/v3"

	"insert/internal/config"
	"insert/internal/handler"

	"github.com/gofiber/fiber/v3/middleware/recover"
)

func main() {

	cfg := config.LoadAllConfig()

    log.Printf("SSH_HOST=%q", cfg.SSHHost)
    log.Printf("SSH_PORT=%q", cfg.SSHPort)
    log.Printf("SSH_USER=%q", cfg.SSHUser)
	db := config.ConnectOracle(cfg)

    app := fiber.New(fiber.Config{
        BodyLimit: 100 * 1024 * 1024, // 100 MB
    })


    app.Use(func(c fiber.Ctx) error {
        log.Println(c.Method(), c.Path())
        return c.Next()
    })

    app.Use(recover.New())

    app.Get("/", func(c fiber.Ctx) error {
        return c.SendFile("./web/index.html")
    })

    app.Post("/api/upload", handler.UploadExcel())

    app.Get("/api/tables",
        handler.GetTables(db),
    )

    app.Get("/api/columns/:table",
        handler.GetColumns(db),
    )

    app.Post("/api/preview", handler.PreviewExcel())

    app.Post("/api/import",
        handler.ImportExcel(db),
    )

    app.Get("/css/*", func(c fiber.Ctx) error {
        return c.SendFile("./web/css/" + c.Params("*"))
    })

    app.Get("/js/*", func(c fiber.Ctx) error {
        return c.SendFile("./web/js/" + c.Params("*"))
    })

	log.Fatal(app.Listen(":3000"))
}