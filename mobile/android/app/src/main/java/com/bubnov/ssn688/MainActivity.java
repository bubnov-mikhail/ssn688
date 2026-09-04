package com.bubnov.ssn688;

import android.os.Bundle;

import androidx.appcompat.app.AppCompatActivity;

import com.bubnov.ssn688.mobile.EbitenView;
import com.bubnov.ssn688.mobile.Mobile;

import go.Seq;

public class MainActivity extends AppCompatActivity {

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        // Application-private files dir for saves / scenarios / import inbox.
        Mobile.setDataRoot(getFilesDir().getAbsolutePath());
        Seq.setContext(getApplicationContext());
        setContentView(R.layout.activity_main);
    }

    private EbitenView getEbitenView() {
        return (EbitenView) this.findViewById(R.id.ebitenview);
    }

    @Override
    protected void onPause() {
        super.onPause();
        this.getEbitenView().suspendGame();
    }

    @Override
    protected void onResume() {
        super.onResume();
        this.getEbitenView().resumeGame();
    }
}
