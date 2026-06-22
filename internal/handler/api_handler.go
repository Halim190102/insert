package handler

import (
	"database/sql"

	"github.com/gofiber/fiber/v3"

	"insert/internal/service"
)

func GetTables(db *sql.DB) fiber.Handler {
	return func(c fiber.Ctx) error {

		tables, err := service.GetOracleTables(db)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(tables)
	}
}

func GetColumns(db *sql.DB) fiber.Handler {
	return func(c fiber.Ctx) error {

		tableName := c.Params("table")

		cols, err := service.GetOracleColumns(
			db,
			tableName,
		)

		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(cols)
	}
}

func UploadExcel() fiber.Handler {
	return func(c fiber.Ctx) error {
		file, err := c.FormFile("excel")
		if err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		path := "./uploads/" + file.Filename
		if err := c.SaveFile(file, path); err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		sheets, err := service.GetSheets(path)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"fileName": file.Filename,
			"sheets":   sheets,
		})
	}
}

type PreviewRequest struct {
	FileName  string `json:"fileName"`
	SheetName string `json:"sheetName"`
}

func PreviewExcel() fiber.Handler {
	return func(c fiber.Ctx) error {
		var req PreviewRequest
		if err := c.Bind().Body(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		if req.FileName == "" {
			return c.Status(400).JSON(fiber.Map{
				"error": "fileName required",
			})
		}

		path := "./uploads/" + req.FileName
		headers, rows, err := service.ReadExcelPreview(path, req.SheetName)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		// Pastikan max 50 baris
		if len(rows) > 50 {
			rows = rows[:50]
		}

		return c.JSON(fiber.Map{
			"fileName":  req.FileName,
			"sheetName": req.SheetName,
			"headers":   headers,
			"rows":      rows,
		})
	}
}

// ImportRequest adalah request body untuk /api/import
type ImportRequest struct {
	FileName        string            `json:"fileName"`
	SheetName       string            `json:"sheetName"`
	Mode            string            `json:"mode"`
	TableName       string            `json:"tableName"`
	SelectedColumns []string          `json:"selectedColumns"` // untuk mode create
	ColumnMap       map[string]string `json:"columnMap"`       // untuk mode insert: excelCol -> oracleCol
}

func ImportExcel(db *sql.DB) fiber.Handler {

	return func(c fiber.Ctx) error {

		var req ImportRequest

		if err := c.Bind().Body(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		if req.FileName == "" {
			return c.Status(400).JSON(fiber.Map{
				"error": "fileName tidak boleh kosong",
			})
		}

		if req.TableName == "" {
			return c.Status(400).JSON(fiber.Map{
				"error": "tableName tidak boleh kosong",
			})
		}

		path := "./uploads/" + req.FileName

		switch req.Mode {

		case "create":
			// Validasi
			if len(req.SelectedColumns) == 0 {
				return c.Status(400).JSON(fiber.Map{
					"error": "pilih minimal satu kolom dari Excel",
				})
			}

			// Buat table baru dengan kolom yang dipilih
			err := service.CreateTable(db, req.TableName, req.SelectedColumns)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{
					"error": err.Error(),
				})
			}

			// Baca semua data Excel
			headers, rows, err := service.ReadExcelFull(path, req.SheetName)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{
					"error": err.Error(),
				})
			}

			// Buat columnMap otomatis: excelCol -> oracleCol (sama, uppercase + underscore)
			autoMap := make(map[string]string)
			for _, sel := range req.SelectedColumns {
				// Cari header yang cocok
				for _, h := range headers {
					if h == sel {
						import_col := ""
						for _, c2 := range []byte(sel) {
							if c2 == ' ' {
								import_col += "_"
							} else {
								import_col += string([]byte{c2})
							}
						}
						autoMap[sel] = toUpper(import_col)
						break
					}
				}
			}

			inserted, skipped, err := service.InsertRows(db, req.TableName, headers, rows, autoMap)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{
					"error": err.Error(),
				})
			}

			return c.JSON(fiber.Map{
				"success":  true,
				"inserted": inserted,
				"skipped":  skipped,
				"message":  "Table berhasil dibuat dan data di-import",
			})

		case "insert":
			// Validasi
			if len(req.ColumnMap) == 0 {
				return c.Status(400).JSON(fiber.Map{
					"error": "column map tidak boleh kosong",
				})
			}

			// Baca semua data Excel
			headers, rows, err := service.ReadExcelFull(path, req.SheetName)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{
					"error": err.Error(),
				})
			}

			inserted, skipped, err := service.InsertRows(db, req.TableName, headers, rows, req.ColumnMap)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{
					"error": err.Error(),
				})
			}

			return c.JSON(fiber.Map{
				"success":  true,
				"inserted": inserted,
				"skipped":  skipped,
				"message":  "Data berhasil di-insert",
			})

		default:
			return c.Status(400).JSON(
				fiber.Map{
					"error": "mode tidak valid. Gunakan 'create' atau 'insert'",
				},
			)
		}
	}
}

// toUpper converts string to uppercase
func toUpper(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			result[i] = c - 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}
