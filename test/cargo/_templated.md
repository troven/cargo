## Template Test

Hello!

I have some Env variables:

* {{ .Env.FOO_BAR }} exported as `FOO_BAR`
* {{ .Env.PWD }}

Also context args such as {{ .App.Foo }} and {{ .App.Bar }}. And collections, see other files.

{{ "hello!" | upper | repeat 5 }}
