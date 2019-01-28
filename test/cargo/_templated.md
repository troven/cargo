## Template Test

Hello! This template has been compiled on {{ .OS.OS }}-{{ .OS.Arch }} using {{ .OS.RuntimeVersion }}

I have some Env variables:

* {{ .Env.FOO_BAR }} exported as `FOO_BAR`
* {{ .Env.PWD }}

Also context args such as {{ .App.Foo }} and {{ .App.Bar }}. And collections, see other files.
