package com.dayboard;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.web.bind.annotation.*;
import org.springframework.http.ResponseEntity;
import org.apache.pdfbox.pdmodel.PDDocument;
import org.apache.pdfbox.text.PDFTextStripper;
import java.io.IOException;
import java.io.InputStream;
import java.util.HashMap;
import java.util.Map;
import java.util.UUID;
import java.util.Date;

@SpringBootApplication
@RestController
@RequestMapping("/api/v1/documents")
public class DocumentProcessor {

    public static void main(String[] args) {
        SpringApplication.run(DocumentProcessor.class, args);
    }

    @PostMapping("/extract-text")
    public ResponseEntity<Map<String, Object>> extractText(@RequestParam("file") MultipartFile file) {
        try {
            String extractedText = "";
            String fileType = file.getContentType();
            
            if ("application/pdf".equals(fileType)) {
                extractedText = extractPdfText(file.getInputStream());
            } else {
                extractedText = "Unsupported file type. PDF extraction supported.";
            }
            
            Map<String, Object> response = new HashMap<>();
            response.put("id", UUID.randomUUID().toString());
            response.put("name", file.getOriginalFilename());
            response.put("type", detectDocumentType(extractedText));
            response.put("dateScanned", new Date());
            response.put("extractedText", extractedText);
            response.put("wordCount", extractedText.split("\\s+").length);
            response.put("keySkills", extractKeySkills(extractedText));
            
            return ResponseEntity.ok(response);
        } catch (Exception e) {
            Map<String, Object> error = new HashMap<>();
            error.put("error", "Failed to process document: " + e.getMessage());
            return ResponseEntity.badRequest().body(error);
        }
    }

    @GetMapping("/demo-documents")
    public ResponseEntity<Object[]> getDemoDocuments() {
        Object[] demoDocuments = new Object[]{
            Map.of(
                "id", UUID.randomUUID().toString(),
                "name", "Resume.pdf",
                "type", "Resume",
                "dateScanned", new Date(),
                "extractedText", "John Doe\nSoftware Engineering Student\nSkills: Java, Swift, Go, PostgreSQL, React\nExperience: iOS Development Intern at TechCorp\nEducation: BS Computer Science, State University\nProjects: DayBoard - Personal productivity app with Go backend and SwiftUI frontend",
                "wordCount", 45,
                "keySkills", new String[]{"Java", "Swift", "Go", "PostgreSQL", "React", "iOS Development"}
            ),
            Map.of(
                "id", UUID.randomUUID().toString(),
                "name", "Offer_Letter.pdf",
                "type", "Offer Letter",
                "dateScanned", new Date(),
                "extractedText", "Dear John,\nWe are pleased to offer you the position of Software Engineering Intern at TechCorp.\nCompensation: $25/hour, 40 hours/week\nStart Date: June 1, 2024\nLocation: Austin, TX\nBenefits: Health insurance, gym membership, free lunch",
                "wordCount", 38,
                "keySkills", new String[]{"Software Engineering", "Austin", "$25/hour"}
            )
        };
        return ResponseEntity.ok(demoDocuments);
    }

    private String extractPdfText(InputStream inputStream) throws IOException {
        try (PDDocument document = PDDocument.load(inputStream)) {
            PDFTextStripper stripper = new PDFTextStripper();
            return stripper.getText(document);
        }
    }

    private String detectDocumentType(String text) {
        String lowerText = text.toLowerCase();
        if (lowerText.contains("resume") || lowerText.contains("experience") || lowerText.contains("skills")) {
            return "Resume";
        } else if (lowerText.contains("offer") || lowerText.contains("compensation") || lowerText.contains("salary")) {
            return "Offer Letter";
        } else if (lowerText.contains("transcript") || lowerText.contains("gpa") || lowerText.contains("course")) {
            return "Transcript";
        } else if (lowerText.contains("cover letter") || lowerText.contains("dear hiring")) {
            return "Cover Letter";
        }
        return "Document";
    }

    private String[] extractKeySkills(String text) {
        String[] commonSkills = {"Java", "Python", "JavaScript", "Swift", "Go", "React", "Node.js", 
                               "PostgreSQL", "MySQL", "AWS", "Docker", "Kubernetes", "Git", "Agile"};
        java.util.List<String> foundSkills = new java.util.ArrayList<>();
        
        for (String skill : commonSkills) {
            if (text.toLowerCase().contains(skill.toLowerCase())) {
                foundSkills.add(skill);
            }
        }
        
        return foundSkills.toArray(new String[0]);
    }
}
