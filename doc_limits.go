package volcenginex

import (
	"path/filepath"
	"strings"
)

const (
	maxDocsPerTask int64 = 10
	maxFileBytes   int64 = 200 << 20
	maxTaskBytes   int64 = 2000 << 20
)

func validateCreateTask(req CreateTaskRequest) error {
	if req.ProjectID <= 0 || req.TaskName == "" {
		return wrapSentinel("DocumentTaskCreate", ErrInvalidArgument, "project ID and task name are required")
	}
	if len(req.Docs) > int(maxDocsPerTask) {
		return formatLimitError(ErrTooManyDocs, "a task supports at most %d documents", maxDocsPerTask)
	}
	var total int64
	for _, doc := range req.Docs {
		if err := validateDocType(doc.Type); err != nil {
			return err
		}
		if doc.Size < 0 {
			return wrapSentinel("DocumentTaskCreate", ErrInvalidArgument, "document size cannot be negative")
		}
		size := doc.Size
		if size == 0 && doc.Content != "" {
			size = int64(len([]byte(doc.Content)))
		}
		if size > maxFileBytes {
			return formatLimitError(ErrFileTooLarge, "document %q exceeds %d bytes", doc.Name, maxFileBytes)
		}
		if total > maxTaskBytes-size {
			return formatLimitError(ErrTaskTooLarge, "task documents exceed %d bytes", maxTaskBytes)
		}
		total += size
	}
	return nil
}

func validateDocType(docType DocType) error {
	switch docType {
	case DocTypeFeishuDoc, DocTypeDocx, DocTypeXlsx, DocTypePptx, DocTypeMd, DocTypePdf:
		return nil
	case DocTypeDoc, DocTypeXls, DocTypePpt, DocTypeCsv, DocTypeFeishuSheet, DocTypeFeishuSlide:
		return formatLimitError(ErrDocTypeUnsupported, "document type %d cannot be parsed online, please save it as .docx/.xlsx/.pptx before translation", docType)
	default:
		return formatLimitError(ErrDocTypeUnsupported, "document type %d is unsupported, please save it as .docx/.xlsx/.pptx before translation", docType)
	}
}

func docTypeFromPath(path string) (DocType, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".doc":
		return DocTypeDoc, validateDocType(DocTypeDoc)
	case ".docx":
		return DocTypeDocx, nil
	case ".xls":
		return DocTypeXls, validateDocType(DocTypeXls)
	case ".xlsx":
		return DocTypeXlsx, nil
	case ".ppt":
		return DocTypePpt, validateDocType(DocTypePpt)
	case ".pptx":
		return DocTypePptx, nil
	case ".csv":
		return DocTypeCsv, validateDocType(DocTypeCsv)
	case ".md", ".markdown":
		return DocTypeMd, nil
	case ".pdf":
		return DocTypePdf, nil
	default:
		return 0, formatLimitError(ErrDocTypeUnsupported, "file extension %q is unsupported, please save it as .docx/.xlsx/.pptx before translation", filepath.Ext(path))
	}
}
