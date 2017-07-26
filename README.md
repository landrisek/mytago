# mytago
Concurrent SQL to CSV write, read, fetch and update

## Usage

```go
package main

import ("github.com/landrisek/mytago")

var builder = mytago.Builder{}.Inject()

func main() {
	callback := func(data map[string]string) string {
		...
		return row
	}
	builder.Table(builder.Config.Tables.MyTable).Select(map[string]string{"alias":"column","alias":"table.column"}).Header(
		"column;column").Where("column", "value").Order("column ASC").Limit(10).Prepare().Write(callback)
}
```