package io.k8s;

import java.io.File;
import java.io.IOException;
import java.util.ArrayList;
import java.util.List;

import net.masterthought.cucumber.Configuration;
import net.masterthought.cucumber.ReportBuilder;
import net.masterthought.cucumber.presentation.PresentationMode;
import net.masterthought.cucumber.sorting.SortingMethod;

public class Main {

    public static void main(String[] args) throws IOException {
        File reportOutputDirectory = new File("/report-output");

        List<String> jsonFiles = new ArrayList<>();
        jsonFiles.add("/input.json");

        String buildNumber = "1";
        String projectName = "Ingress Conformance Test";

        Configuration configuration = new Configuration(reportOutputDirectory, projectName);
        configuration.setBuildNumber(buildNumber);
        configuration.addClassifications("Release", "1.19");
        configuration.setSortingMethod(SortingMethod.NATURAL);
        configuration.setTrendsStatsFile(new File("/report-output/trends.json"));

        ReportBuilder reportBuilder = new ReportBuilder(jsonFiles, configuration);
        reportBuilder.generateReports();
    }
}
