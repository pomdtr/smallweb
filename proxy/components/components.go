package components

import (
	"fmt"

	g "github.com/maragudk/gomponents"
	c "github.com/maragudk/gomponents/components"

	//lint:ignore ST1001 fine for gomponents
	. "github.com/maragudk/gomponents/html"
)

func Home(username string, apps []string) g.Node {
	return c.HTML5(c.HTML5Props{
		Title:       "Home",
		Description: "Welcome to smallweb",
		Head: []g.Node{
			Link(Rel("stylesheet"), Href("https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.min.css")),
		},
		Body: []g.Node{
			Main(
				Class("container"),
				Ul(
					g.Map(
						apps, func(app string) g.Node {
							return Li(
								A(
									Href(fmt.Sprintf("https://%s-%s.smallweb.run", app, username)),
									g.Text(app),
								),
							)
						},
					)...,
				),
			),
		},
	})
}
