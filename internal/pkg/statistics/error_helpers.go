package statistics

import "time"

// === ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ДЛЯ АНАЛИЗА ОШИБОК ===

// getDurationCategory возвращает категорию длительности
func getDurationCategory(duration time.Duration) string {
	if duration < 1*time.Second {
		return "instant"
	} else if duration < 5*time.Second {
		return "fast"
	} else if duration < 30*time.Second {
		return "normal"
	} else if duration < 60*time.Second {
		return "slow"
	} else {
		return "timeout"
	}
}

// getLikelyCause возвращает вероятную причину ошибки HTTP запроса
func getLikelyCause(path string, duration time.Duration) string {
	if duration < 100*time.Millisecond {
		return "Connection rejected, service unavailable, or validation error"
	} else if duration > 60*time.Second {
		return "Timeout - downstream service hanging or processing large data"
	} else if path == "/api/v1/docx" {
		return "DOCX generation error or Python script failure"
	} else if path == "/generate-pdf" {
		return "PDF conversion error in Gotenberg"
	} else {
		return "Unknown service error"
	}
}

// getDocxErrorCause возвращает вероятную причину ошибки DOCX
func getDocxErrorCause(duration time.Duration) string {
	if duration < 1*time.Second {
		return "Python script crashed, missing template, or file system error"
	} else if duration < 5*time.Second {
		return "Data validation error, template parsing issue, or invalid input"
	} else if duration > 30*time.Second {
		return "Large data processing, Python hung, or memory exhaustion"
	} else {
		return "Logic error in template processing or data transformation"
	}
}

// getDocxTroubleshooting возвращает рекомендации по устранению ошибки DOCX
func getDocxTroubleshooting(duration time.Duration) string {
	if duration < 1*time.Second {
		return "Check: Python environment, template file exists, write permissions, disk space"
	} else if duration < 5*time.Second {
		return "Check: Input data format, template syntax, required fields presence"
	} else if duration > 30*time.Second {
		return "Check: Data size, Python memory limits, process timeouts, system resources"
	} else {
		return "Check: Template logic, data mapping, Python script logs"
	}
}

// getGotenbergErrorCause возвращает вероятную причину ошибки Gotenberg
func getGotenbergErrorCause(duration time.Duration) string {
	if duration < 2*time.Second {
		return "Gotenberg service unavailable, network error, or invalid request format"
	} else if duration > 120*time.Second {
		return "LibreOffice hung, complex document, or memory/resource exhaustion"
	} else if duration > 60*time.Second {
		return "Complex document conversion, fonts missing, or heavy processing"
	} else {
		return "Document format error, corrupted file, or conversion logic failure"
	}
}

// getGotenbergTroubleshooting возвращает рекомендации по устранению ошибки Gotenberg
func getGotenbergTroubleshooting(duration time.Duration) string {
	if duration < 2*time.Second {
		return "Check: Gotenberg service health, network connectivity, request format"
	} else if duration > 120*time.Second {
		return "Check: LibreOffice processes, system memory, conversion timeouts, restart Gotenberg"
	} else if duration > 60*time.Second {
		return "Check: Document complexity, fonts availability, processing limits"
	} else {
		return "Check: Document format, file integrity, supported features"
	}
}
