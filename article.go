package client

// Visibility describes who can view the given item (e.g. help article)
type Visibility string

const (
	// VisibilityPublic means that the item is available to the general
	// public. For example, it is published on a public website.
	VisibilityPublic Visibility = "public"

	// VisibilityUsers means that the item is only available to the
	// company's customers. For example, it is only accessible via an
	// app after sign-up.
	VisibilityUsers Visibility = "users"

	// VisibilityInternal means that the item is only available to
	// the company's employees. For example, it is a procedure or SOP
	// that customers do not have access to.
	VisibilityInternal Visibility = "internal"

	// VisibilityUnknown means the visibility could not be determined. It only
	// appears on topics that Gradient Labs read from a support platform.
	VisibilityUnknown Visibility = "unknown"
)

// PublicationStatus describes the status of a help article.
type PublicationStatus string

const (
	// PublicationStatusDraft means that the article is being written or
	// edited and is not published.
	PublicationStatusDraft PublicationStatus = "draft"

	// PublicationStatusPublished means that the article is published.
	PublicationStatusPublished PublicationStatus = "published"

	// PublicationStatusDeleted means that the article was removed from your
	// help centre.
	PublicationStatusDeleted PublicationStatus = "deleted"

	// PublicationStatusExcluded means that the article still exists, but the
	// AI agent should not use it.
	PublicationStatusExcluded PublicationStatus = "excluded"
)
