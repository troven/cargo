## cargo  by troven

<img src="docs/cargo.png" width="300px" />

## Cargo Templates

Cargo offers a simple but powerful tool to automate common build jobs. However you want. With minimal opinions - except for those inherent to the Go Template library. 

A Cargo package contain files and templates. Templates are text files that include processing instructions. The simplest Cargo package is a directory with a single file or folder in it.

Templates can be used to create almost anything. 

## Cargo Rendering

Here is an example that creates a Cargo catalog page in HTML.

```
cat ./local/tests/_test..html

<html><body>
	<h1>{{.Cargo.Name}} by {{.Cargo.Author.Name}}<h1>
</body></html>
```

It only uses `{{.Cargo}}` which is injected into every Cargo context.

To run build our documentation, we can simply:

```
	cargo run local/tests/ ./local/build
```


## Custom Context

Let's create a custom Context so cargo can resolve the {{.Page.title}} reference.

We create a page.yaml and specify any meta-data we need about our web page.

```
cat ./local/page.yaml

title: Hello World
description: This is not displayed
```

And modify our HTML to something like:

```
<h1>{{.Cargo.Name}} by {{.Cargo.Author.Name}}<h1>
<h2>{{.Page.Title}}<h1>
```

We need to pass in our Page context, like this:

```
cargo run ./local/tests/ ./local/build --context Page=page.yaml
```

## Making Decisions

You can use `if` expressions in templates. 

```
{{if .Cargo.Name}} 
	Our Cargo is {{ .Name }} 
{{else}}
	Oops! Cargo.Name is missing 
{{end}}
```

Note that `else` and `else if` are supported. 

## Looping Around

With  the `range` expression you can loop through a collection. A range actions is defined using the `{{range .Items}} ... {{end}}` construct.

If your collection is a simple type, you can refer to the value using the {{ . }} parameter. 


