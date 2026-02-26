package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "password" // Matches the Docker command
	dbname   = "postgres" // Default database created by Docker
)

// ─────────────────────────────────────────────
// B-TREE INDEX DEMO
//
// B-tree is PostgreSQL's default index type.
// Best for: equality (=), range (<, >, BETWEEN),
//           ORDER BY, and LIKE 'prefix%'.
//
// Internally it is a balanced tree where every
// leaf page holds sorted key→ctid (row pointer)
// pairs. Lookups are O(log n).
// ─────────────────────────────────────────────

func setupBtree(db *sql.DB) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS products (
			id      SERIAL PRIMARY KEY,
			name    TEXT    NOT NULL,
			price   NUMERIC NOT NULL
		)`)
	if err != nil {
		log.Fatal("create products table:", err)
	}

	// B-tree index on price — enables fast range scans like
	//   WHERE price BETWEEN 10 AND 50
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_products_price_btree
		ON products USING btree (price)`)
	if err != nil {
		log.Fatal("create btree index:", err)
	}

	fmt.Println("[B-tree] Table and index ready.")
}

func insertBtree(db *sql.DB) {
	rows := []struct {
		name  string
		price float64
	}{
		{"Widget A", 9.99},
		{"Widget B", 24.50},
		{"Widget C", 49.00},
		{"Gadget X", 99.95},
		{"Gadget Y", 199.00},
	}

	for _, r := range rows {
		_, err := db.Exec(
			`INSERT INTO products (name, price) VALUES ($1, $2)
			 ON CONFLICT DO NOTHING`, // idempotent re-runs
			r.name, r.price,
		)
		if err != nil {
			log.Fatal("insert product:", err)
		}
	}
	fmt.Println("[B-tree] Inserted 5 product rows.")
}

func queryBtree(db *sql.DB) {
	// PostgreSQL will use idx_products_price_btree here because
	// the WHERE clause is a range predicate on an indexed column.
	fmt.Println("\n[B-tree] Products priced between $10 and $100:")
	rowset, err := db.Query(
		`SELECT id, name, price FROM products
		 WHERE price BETWEEN 10 AND 100
		 ORDER BY price`)
	if err != nil {
		log.Fatal("query btree:", err)
	}
	defer rowset.Close()

	for rowset.Next() {
		var id int
		var name string
		var price float64
		rowset.Scan(&id, &name, &price)
		fmt.Printf("  id=%d  %-12s  $%.2f\n", id, name, price)
	}
}

// ─────────────────────────────────────────────
// GIN INDEX DEMO
//
// GIN = Generalized Inverted Index.
// Best for: full-text search (tsvector), JSONB
//           containment (@>, ?), and array overlap (&&).
//
// Internally GIN builds a posting list:
//   token → [list of row pointers that contain it]
// A single document's tokens are "inverted" into
// many index entries, which is why it is called
// an inverted index.  Lookups are O(log n) per token.
// ─────────────────────────────────────────────

func setupGIN(db *sql.DB) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS articles (
			id      SERIAL PRIMARY KEY,
			title   TEXT NOT NULL,
			body    TEXT NOT NULL,
			tsv     TSVECTOR   -- pre-computed full-text vector
		)`)
	if err != nil {
		log.Fatal("create articles table:", err)
	}

	// GIN index on the tsvector column — enables fast full-text
	// queries like  WHERE tsv @@ to_tsquery('postgres')
	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_articles_tsv_gin
		ON articles USING gin (tsv)`)
	if err != nil {
		log.Fatal("create GIN index:", err)
	}

	fmt.Println("[GIN]    Table and index ready.")
}

func insertGIN(db *sql.DB) {
	articles := []struct {
		title string
		body  string
	}{
		{
			"Introduction to PostgreSQL",
			"PostgreSQL is a powerful open-source relational database system.",
		},
		{
			"Understanding Indexes",
			"B-tree indexes are great for sorted and range queries in SQL databases.",
		},
		{
			"Full-Text Search with GIN",
			"GIN indexes allow fast full-text search over large text documents.",
		},
		{
			"Go and Postgres",
			"Using the lib/pq driver you can connect Go programs to a Postgres database.",
		},
		{
			"Kubernetes Operators",
			"Operators extend Kubernetes with custom controllers that manage complex workloads.",
		},
	}

	for _, a := range articles {
		// to_tsvector parses the text into lexemes (stemmed tokens)
		// and stores them in the tsv column so GIN can index them.
		_, err := db.Exec(`
			INSERT INTO articles (title, body, tsv)
			VALUES ($1, $2, to_tsvector('english', $1 || ' ' || $2))
			ON CONFLICT DO NOTHING`,
			a.title, a.body,
		)
		if err != nil {
			log.Fatal("insert article:", err)
		}
	}
	fmt.Println("[GIN]    Inserted 5 article rows.")
}

func queryGIN(db *sql.DB) {
	// @@ is the full-text match operator.
	// to_tsquery turns the search string into a query tree.
	// PostgreSQL uses idx_articles_tsv_gin to resolve it fast.
	term := "database"
	fmt.Printf("\n[GIN]    Articles matching '%s':\n", term)

	rowset, err := db.Query(`
		SELECT id, title
		FROM   articles
		WHERE  tsv @@ to_tsquery('english', $1)
		ORDER  BY id`, term)
	if err != nil {
		log.Fatal("query GIN:", err)
	}
	defer rowset.Close()

	for rowset.Next() {
		var id int
		var title string
		rowset.Scan(&id, &title)
		fmt.Printf("  id=%d  %s\n", id, title)
	}
}

// ─────────────────────────────────────────────
// Entry point wiring everything together
// ─────────────────────────────────────────────

func Pg_Conn() {
	psqlInfo := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal("failed to open DB:", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatal("failed to ping DB (is Docker running?):", err)
	}
	fmt.Println("Successfully connected to Postgres!")
	fmt.Println()

	// ── B-tree demo ──────────────────────────
	setupBtree(db)
	insertBtree(db)
	queryBtree(db)

	fmt.Println()

	// ── GIN demo ─────────────────────────────
	setupGIN(db)
	insertGIN(db)
	queryGIN(db)
}
