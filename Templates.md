## cargo  by troven

<img src="docs/cargo.png" width="300px" />

## Cargo Templates

Cargo packages contain files and templates. Templates are text files that include processing instructions.

Templates can create almost anything. Here is an example that creates some HTML

```
cat ./src/_test..html

<html><body>
	<h1>{{.Page.title}}<h1>
</body></html>
```

We will need to provide a Context so cargo can resolve the {{.Page.title}} reference.

We create a page.yaml and specify any meta-data we need about our web page.

```
cat page.yaml

title: Hello World
description: This is not displayed
```

To turn our template into a real web page, we can now type:

```
cargo run ./src ./published --context Page=page.yaml
```



