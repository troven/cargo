## cargo  by troven

<img src="docs/cargo.png" width="300px" />

## Cargo Templates

Cargo packages contain files and templates. Templates are text files that include processing instructions.

Templates can create almost anything. Here is an example that creates some HTML

src/_test..html
```
	<html><body>
		<h1>{{.Page.title}}<h1>
	</body></html>
```

Cargo turns it into a real HTML page:

```
cargo run ./src ./published --context Page=page.yaml
```

Except we forgot the page.yaml :-)

page.yaml
```
title: Hello World
```



