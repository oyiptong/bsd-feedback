package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/gin-contrib/multitemplate"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	DEFAULT_PORT = 8080
)

type serverConfig struct {
	Port       int
	GCPProject string
}

var config = serverConfig{
	Port:       DEFAULT_PORT,
	GCPProject: "",
}

func init() {
	envPort := os.Getenv("PORT")
	if envPort != "" {
		if p, err := strconv.Atoi(envPort); err != nil && p > 0 {
			config.Port = p
		}
	}
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		projectID = os.Getenv("GCP_PROJECT")
	}

	config.GCPProject = projectID
}

type School struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type Concern struct {
	Title       string        `json:"title"`
	Description template.HTML `json:"description"`
	Slug        string        `json:"slug"`
}

type LetterForm struct {
	ID              string   `form:"id" json:"id" binding:"required"`
	Email           string   `form:"email" json:"email" binding:"required"`
	Name            string   `form:"name" json:"name" binding:"required"`
	NumChildren     int      `form:"children" json:"children" binding:"required"`
	Schools         []string `form:"schools" json:"schools" binding:"required"`
	Concerns        []string `form:"concerns" json:"concerns" binding:"required"`
	SchoolsChecked  map[string]bool
	ConcernsChecked map[string]bool
	FreeForm        string `form:"freeform" json:"freeform"`
}

type Letter struct {
	Key          *datastore.Key `datastore:"__key__" json:"-"`
	ID           string         `json:"id"`
	Email        string         `json:"email"`
	Name         string         `json:"name"`
	NumChildren  int            `json:"numChildren"`
	Schools      []School       `json:schools`
	Concerns     []Concern      `json:concerns`
	FreeForm     string         `json:freeForm`
	CreatedAt    time.Time      `json:createdAt`
	SendCount    int            `json:sendCount`
	SentReceipts []time.Time    `json:sentReceipts`
}

var letterDB = map[string]Letter{}
var schoolDB = map[string]School{}
var concernDB = map[string]Concern{}

func formToLetter(form LetterForm) Letter {
	var letter Letter
	letter.ID = form.ID
	letter.Email = form.Email
	letter.Name = form.Name
	letter.NumChildren = form.NumChildren
	letter.FreeForm = form.FreeForm
	for _, slug := range form.Schools {
		letter.Schools = append(letter.Schools, schoolDB[slug])
	}
	for _, slug := range form.Concerns {
		letter.Concerns = append(letter.Concerns, concernDB[slug])
	}
	return letter
}

func letterToForm(letter Letter) LetterForm {
	var form LetterForm
	form.SchoolsChecked = make(map[string]bool)
	form.ConcernsChecked = make(map[string]bool)
	form.ID = letter.ID
	form.Email = letter.Email
	form.Name = letter.Name
	form.NumChildren = letter.NumChildren
	form.FreeForm = letter.FreeForm
	for _, school := range letter.Schools {
		// Replace slug with name.
		form.Schools = append(form.Schools, school.Slug)
		form.SchoolsChecked[school.Slug] = true
	}
	for _, concern := range letter.Concerns {
		form.Concerns = append(form.Concerns, concern.Slug)
		form.ConcernsChecked[concern.Slug] = true
	}
	return form
}

func persistLetter(ctx context.Context, letter Letter, client *datastore.Client) error {
	newKey := datastore.NameKey("Letter", letter.ID, nil)
	letter.CreatedAt = time.Now()

	_, err := client.Put(ctx, newKey, &letter)
	if err != nil {
		log.Println("Failed to persist letter: ", err)
		return err
	}
	return nil
}

func getLetter(ctx context.Context, id string, client *datastore.Client) (Letter, error) {
	var letter Letter

	k := datastore.NameKey("Letter", id, nil)
	if err := client.Get(ctx, k, &letter); err != nil {
		return letter, err
	}

	return letter, nil
}

func main() {
	ctx := context.Background()
	dsClient, err := datastore.NewClient(ctx, config.GCPProject)
	if err != nil {
		log.Fatalf("Failed to connect to Datastore: %v", err)
	}

	r := gin.Default()
	r.SetFuncMap(template.FuncMap{
		"plus1": func(x int) int {
			return x + 1
		},
	})
	r.HTMLRender = loadTemplates("./templates")
	r.Use(static.Serve("/static/", static.LocalFile("./static", false)))

	var schools = []School{
		School{"Burlingame Intermediate School", "bis"},
		School{"Franklin Elementary School", "franklin"},
		School{"Hoover Elementary School", "hoover"},
		School{"Lincoln Elementary School", "lincoln"},
		School{"McKinley Elementary School", "mckinley"},
		School{"Roosevelt Elementary School", "roosevelt"},
		School{"Washington Elementary School", "washington"},
	}
	for _, school := range schools {
		schoolDB[school.Slug] = school
	}

	var concerns = []Concern{
		Concern{
			"I have no concerns",
			template.HTML("I actually have no concerns. I'm happy with the direction BSD has taken and every step of the plan is clear to me."),
			"no-concerns",
		},
		Concern{
			"Have our teachers been heard?",
			template.HTML("How did the administration and teachers work collaboratively on the proposed plans? How do we (as parents) know teachers approve and support the hybrid model and its potential for success for students continuing to learn remotely and those returning to the classroom? What surveys or polls have the district used to gage how teachers are feeling about the potential reopening of schools? Please provide details on why the plan has been laid out the way it is presented so that we can have a better understanding of the considerations, safety precautions and programs that are being put in place."),
			"teachers-heard",
		},
		Concern{
			"What will Distance Learning Look Like?",
			template.HTML("Why did the school district decide to open with a hybrid model while infection rate is increasing, instead of starting with distance learning and slowly easing into the hybrid model with a phased approach like other districts within the county? The administration mentions SB 98 limits the district’s ability to offer distance learning as a stand-alone model except on a per-parent request for medical necessity or if determined necessary by a local health agency. How then are other SMC districts, such as Millbrae, Menlo Park and SMFCSD, planning to start the year 100% distance learning then phasing in hybrid? With the inevitability of a second Covid wave and the annual flu season coming, distance learning will likely be with us in some form for the foreseeable future. What has the district learned from the spring to improve upon the distance learning curriculum and experience to have consistency for all BSD students?"),
			"distance-learning",
		},
		Concern{
			"What is the Safety Protocol for Classroom Instruction?",
			template.HTML("What rigorous safety protocols are being put in place for students and teachers returning to the classroom? Will they be tested before the first day of school? How often will they be tested thereafter? Will there be daily temperature checks? How often will the school be disinfected and how will this be adequately done without hiring additional custodial personnel? Will there be recess, PE, socializing or collaborating? If there is no social/emotional learning, what is the value?"),
			"safety-protocol",
		},
		Concern{
			"What Happens in an Outbreak?",
			template.HTML("If we return to classroom instruction of any kind, what happens when a student, teacher, school staffer or parent tests positive for Covid-19? What happens if they show symptoms and get tested, but have to wait several days for the results? What happens to the other students in that class? What about the other cohort that shares the classroom? If the teacher contracts Covid, will there be a replacement ready? Does the district plan to hire extra subs?"),
			"outbreak-plans",
		},
	}
	for _, concern := range concerns {
		concernDB[concern.Slug] = concern
	}

	letterDB["foo"] = Letter{
		ID:          "foo",
		Email:       "foo@foo.com",
		Name:        "Froober Goober",
		NumChildren: 3,
		Schools:     schools[5:],
		Concerns:    concerns[2:],
	}

	r.GET("/", func(c *gin.Context) {
		formUuid, err := uuid.NewRandom()
		if err != nil {
			c.HTML(http.StatusInternalServerError, "form.html", gin.H{})
		}
		form := LetterForm{ID: formUuid.String()}

		c.HTML(http.StatusOK, "form.html", gin.H{
			"title":    "BSD Feedback Form",
			"schools":  schools,
			"concerns": concerns,
			"form":     form,
		})
	})

	r.GET("/list-all", func(c *gin.Context) {
		var letters []Letter
		q := datastore.NewQuery("Letter")
		if _, err := dsClient.GetAll(ctx, q, &letters); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, letters)
	})

	r.POST("/", func(c *gin.Context) {
		var form LetterForm
		if err := c.ShouldBind(&form); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		letter := formToLetter(form)
		if err := persistLetter(ctx, letter, dsClient); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Redirect(http.StatusFound, fmt.Sprintf("/letter/preview/%s", letter.ID))
	})

	r.GET("/letter/preview/:id", func(c *gin.Context) {
		urlID := c.Param("id")

		if urlID == "foo" {
			foo := letterDB["foo"]

			c.HTML(http.StatusOK, "preview.html", gin.H{
				"title":  "TEST PAGE DO NOT SHARE",
				"letter": foo,
			})
			return
		}

		letter, err := getLetter(ctx, urlID, dsClient)

		if err == datastore.ErrNoSuchEntity {
			c.HTML(http.StatusNotFound, "404.html", gin.H{})
			return
		}

		if err != nil {
			c.HTML(http.StatusInternalServerError, "500.html", gin.H{"error": err.Error()})
			return
		}

		c.HTML(http.StatusOK, "preview.html", gin.H{
			"title":  "Letter to the BSD Superintendent",
			"letter": letter,
		})
	})

	r.GET("/letter/edit/:id", func(c *gin.Context) {
		urlID := c.Param("id")

		if urlID == "foo" {
			foo := letterDB["foo"]

			c.HTML(http.StatusOK, "preview.html", gin.H{
				"title":  "Preview",
				"letter": foo,
			})
			return
		}

		letter, err := getLetter(ctx, urlID, dsClient)
		if err == datastore.ErrNoSuchEntity {
			c.HTML(http.StatusNotFound, "404.html", gin.H{})
			return
		}

		form := letterToForm(letter)

		c.HTML(http.StatusOK, "form.html", gin.H{
			"title":    "Edit",
			"schools":  schools,
			"concerns": concerns,
			"form":     form,
		})
	})

	r.POST("/letter/record-send/:id", func(c *gin.Context) {
		urlID := c.Param("id")

		letter, err := getLetter(ctx, urlID, dsClient)

		if err == datastore.ErrNoSuchEntity {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		letter.SendCount += 1
		letter.SentReceipts = append(letter.SentReceipts, time.Now())

		if err := persistLetter(ctx, letter, dsClient); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, "ok")
	})

	r.Run(fmt.Sprintf(":%d", config.Port))
}

func loadTemplates(templatesDir string) multitemplate.Renderer {
	r := multitemplate.NewRenderer()

	// Custom template functions.
	funcMap := template.FuncMap{
		"plus1": func(x int) int {
			return x + 1
		},
		"isChecked": func(item string, checkedMap map[string]bool) bool {
			val, found := checkedMap[item]
			return found && val
		},
	}

	layouts, err := filepath.Glob(templatesDir + "/layouts/*.html")
	if err != nil {
		panic(err.Error())
	}

	includes, err := filepath.Glob(templatesDir + "/includes/*.html")
	if err != nil {
		panic(err.Error())
	}

	// Generate our templates map from our layouts/ and includes/ directories.
	for _, include := range includes {
		layoutCopy := make([]string, len(layouts))
		copy(layoutCopy, layouts)
		files := append(layoutCopy, include)
		r.AddFromFilesFuncs(filepath.Base(include), funcMap, files...)
	}
	return r
}
