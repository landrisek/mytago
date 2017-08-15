package mytago

import ("bufio"
	"database/sql"
	"github.com/jinzhu/configor"
	_ "github.com/go-sql-driver/mysql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"regexp"
	"sync")

type Builder struct {
	add map[string]interface{}
	database *sql.DB
	columns map[string]string
	Config struct {
		Api string
		Database struct {
			Host     string
			Name     string
			User     string
			Password string
			Port     int
		}
		Concurrency int
		File string
		Separator string
		Tables struct {
			Orders  string
			Products  string
			Vouchers string
			Transactions  string
		}
	}
	group string
	header string
	limit int
	leftJoin []string
	order string
	rows *sql.Rows
	table string
	where map[string]interface{}
	update map[string]interface{}
	query string
}

func(builder Builder) Add(column string, value interface{}) Builder {
	builder.add[column] = value
	return builder
}

func(builder Builder) Inject(config string) Builder {
	configor.Load(&builder.Config, config)
	database := builder.Config.Database
	builder.database, _ = sql.Open("mysql", database.User + ":" + database.Password  + "@tcp(" + database.Host +
		":" + strconv.Itoa(database.Port) + ")/" + database.Name + "?parseTime=true&collation=utf8_czech_ci")
	return builder
}

func(builder Builder) Fetch() []map[string]string {
	results := make([]map[string]string, 0)
	if 0 == len(builder.query) {
		log.Panic("Query is empty. Did you call Builder:prepare method?")
	}
	rows, _ := builder.database.Query(builder.query)
	columns, _ := rows.Columns()
	data := make([][]byte, len(columns))
	pointers := make([]interface{}, len(columns))
	for i := range data {
		pointers[i] = &data[i]
	}
	for rows.Next() {
		row := make(map[string]string, 0)
		err := rows.Scan(pointers...)
		if err != nil {
			log.Panic(err)
		}
		for key := range data {
			row[columns[key]] = string(data[key])
		}
		results = append(results, row)
	}
	return results
}

func(builder Builder) Group(group string) Builder {
	builder.group = group
	return builder
}

func(builder Builder) Header(header string) Builder {
	builder.header = header
	return builder
}

func(builder Builder) LeftJoin(join string) Builder {
	builder.leftJoin = append(builder.leftJoin, join)
	return builder
}

func(builder Builder) Limit(limit int) Builder {
	builder.limit = limit
	return builder
}

func(builder Builder) Order(order string) Builder {
	builder.order = order
	return builder
}

func(builder Builder) Prepare() Builder {
	builder.query = "SELECT "
	for alias, column := range builder.columns {
		builder.query += column + " AS " + alias + ", "
	}
	builder.query = strings.TrimRight(builder.query, ", ")
	builder.query += " FROM " +  builder.table
	for _, join := range builder.leftJoin {
		builder.query += " LEFT JOIN " + join + " "
	}
	if len(builder.where) > 0 {
		builder.query += " WHERE "
	}
	for column, value := range builder.where {
		if 0 == len(value.(string)) {
			builder.query += column + " AND "
		} else if false == regexp.MustCompile("//=|/>|/<|/sLIKE|/sIN|/sIS|/sNOT|/sNULL|/sNULL/").Match([]byte(column)) {
			builder.query += column + " '" + value.(string) + "' AND "
		} else {
			builder.query += column + " = '" + value.(string) + "' AND "
		}
	}
	builder.query = strings.TrimRight(builder.query, " AND")
	if len(builder.group) > 0 {
		builder.query += " GROUP BY " + builder.group + " "
	}
	if len(builder.order) > 0 {
		builder.query += " ORDER BY " + builder.order
	}
	if builder.limit > 0 {
		builder.query += " LIMIT " + strconv.Itoa(builder.limit) + " "
	}
	return builder
}

func(builder Builder) Read() Builder {
	queue := make(chan string)
	complete := make(chan bool)
	go func() {
		file, err := os.Open(builder.Config.File)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			queue <- scanner.Text()
		}
		close(queue)
	}()
	for i := 0; i < builder.Config.Concurrency; i++ {
		go read(queue, complete, builder.Config.Separator)
	}
	for i := 0; i < builder.Config.Concurrency; i++ {
		<-complete
	}
	return builder
}

func read(queue chan string, complete chan bool, separator string) {
	for line := range queue {
		values := strings.Split(line, separator)
		fmt.Print(values[5], "\n")
	}
	complete <- true
}

func(builder Builder) Select(columns map[string]string) Builder {
	builder.columns = columns
	return builder
}

func(builder Builder) Table(table string) Builder {
	builder.table = table
	return builder
}

func(builder Builder) Update(column string, value interface{}) Builder {
	builder.update[column] = value
	return builder
}

func(builder Builder) Where(column string, value interface{}) Builder {
	if 0 == len(builder.where) {
		builder.where = make(map[string]interface{})
	}
	builder.where[column] = value
	return builder
}

func(builder Builder) Write(callback func(row map[string]string) string) Builder {
	queue := make(chan string)
	complete := make(chan bool)
	file, err := os.Create(builder.Config.File)
	file.Write([]byte(builder.header + "\n"))
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	go func() {
		rows, _ := builder.database.Query(builder.query)
		columns, _ := rows.Columns()
		data := make([][]byte, len(columns))
		pointers := make([]interface{}, len(columns))
		for i := range data {
			pointers[i] = &data[i]
		}
		for rows.Next() {
			row := make(map[string]string, 0)
			err := rows.Scan(pointers...)
			if err != nil {
				fmt.Print(err)
			}
			for key := range data {
				row[columns[key]] = string(data[key])
			}
			queue <- callback(row)
		}
		defer builder.database.Close()
		close(queue)
	}()
	var waitGroup sync.WaitGroup
	waitGroup.Add(builder.Config.Concurrency)
	for i := 0; i < builder.Config.Concurrency; i++ {
		go write(&waitGroup, queue, complete, file)
	}
	for i := 0; i < builder.Config.Concurrency; i++ {
		<-complete
	}
	waitGroup.Wait()
	return builder
}

func write(waitGroup *sync.WaitGroup, queue chan string, complete chan bool, file *os.File) {
	defer waitGroup.Done()
	for row := range queue {
		file.Write([]byte(row + "\n"))
	}
	complete <- true
}